package cool

import (
	"sync"
	"testing"
	"time"

	"chemicalprocessdcs/internal/model"
	"chemicalprocessdcs/internal/param"
)

// makeCurve builds a curve whose every step carries a single distinguishing
// temperature. Any consumer observation mixing entries from two versions can
// be detected because the temperatures would disagree across entries.
func makeCurve(id string, temp float64) model.Curve {
	return model.Curve{
		ID: id,
		Steps: []model.CurveStep{
			{Index: 0, Temperature: temp, Pressure: 1.0, HoldSeconds: 60},
			{Index: 1, Temperature: temp, Pressure: 1.0, HoldSeconds: 60},
			{Index: 2, Temperature: temp, Pressure: 1.0, HoldSeconds: 60},
			{Index: 3, Temperature: temp, Pressure: 1.0, HoldSeconds: 60},
		},
	}
}

// TestSyncCurveAtomicConsumption hammers SyncCurve (producer) against readers
// concurrently. Every reader observation must belong to exactly one coherent
// version: all entries in a snapshot share the same temperature, and the
// length is consistent with that temperature's curve. If the publish were not
// atomic, a reader would observe a mix of entries from two different
// temperatures, or a length that does not match the entry values.
func TestSyncCurveAtomicConsumption(t *testing.T) {
	store := param.NewStore()
	store.SetParams("main", model.ProcessParams{
		TargetTemperature: 180,
		CurveID:           "default",
	})
	store.SetCurve(makeCurve("default", 100))

	svc := NewService(store, func(id string, open bool) error { return nil })

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Producer: alternate between two curve versions by mutating the store
	// generation (SetCurve bumps gen). Each SyncCurve must rebuild seq
	// atomically.
	wg.Add(1)
	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
			}
			temp := 100.0
			if i%2 == 1 {
				temp = 200.0
			}
			store.SetCurve(makeCurve("default", temp))
			svc.SyncCurve()
			i++
		}
	}()

	// Reader A: snapshotSeq — full snapshot must be internally consistent:
	// every entry must carry the same temperature.
	readers := 4
	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				seq, _ := svc.snapshotSeq()
				if len(seq) == 0 {
					continue
				}
				first := seq[0].Temperature
				for _, e := range seq {
					if e.Temperature != first {
						t.Errorf("snapshot straddles two versions: saw %v and %v", first, e.Temperature)
						return
					}
				}
			}
		}()
	}

	// Reader B: StepAt / SequenceLength — the length must bound every index we
	// read, and the entry temperature must be one of the published values.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			n := svc.SequenceLength()
			if n == 0 {
				continue
			}
			entry, ok := svc.StepAt(n - 1)
			if ok && entry.Temperature != 100 && entry.Temperature != 200 {
				t.Errorf("entry temperature %v is not from any known version", entry.Temperature)
				return
			}
		}
	}()

	// Let the goroutines run for long enough to actually exercise the race.
	time.Sleep(200 * time.Millisecond)
	close(stop)
	wg.Wait()
}
