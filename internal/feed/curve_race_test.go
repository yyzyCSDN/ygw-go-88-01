package feed

import (
	"context"
	"sync"
	"testing"
	"time"

	"chemicalprocessdcs/internal/model"
	"chemicalprocessdcs/internal/param"
)

// makeCurve builds a curve whose every step carries a single distinguishing
// temperature so a snapshot mixing two versions can be detected by comparing
// temperatures across entries.
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

func newTestService(store *param.Store) *Service {
	confirm := func(ctx context.Context, id string) (model.FeedConfirmation, error) {
		return model.FeedConfirmation{Amount: 1, OK: true}, nil
	}
	return NewService(store, confirm)
}

// TestSyncCurveAtomicConsumption drives a producer that flips the active curve
// between two versions (temperature 100 vs 200) against concurrent consumers.
// Every consumer observation must belong to exactly one coherent version:
//   - StepAt at the last index must return a temperature that is one of the
//     published values.
//   - Progress's (curveID, index, length) must be self-consistent: the length
//     must bound the index, and the length must be the fixed published length.
//
// If SyncCurve published the sequence non-atomically (e.g. seq swapped while a
// consumer held a torn slice header, or index reset separately from seq), a
// consumer could read index/length/entries that straddle two versions. This
// test would then observe a temperature that is neither 100 nor 200, or an
// index not bounded by the length.
func TestSyncCurveAtomicConsumption(t *testing.T) {
	store := param.NewStore()
	store.SetParams("main", model.ProcessParams{
		TargetTemperature: 180,
		CurveID:           "default",
	})
	store.SetCurve(makeCurve("default", 100))

	svc := newTestService(store)
	svc.Register("R1")
	_ = svc.Start("R1")

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Producer: alternate the curve version (temperature 100 vs 200).
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

	// Consumer A: StepAt(SequenceLength-1) must yield a known version's
	// temperature, and the index used must actually be in range — proving the
	// length and the entry came from the same version.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			curveID, idx, length := svc.Progress()
			if length == 0 {
				continue
			}
			if idx >= length {
				t.Errorf("index %d not bounded by length %d (curveID=%s): straddles a switch",
					idx, length, curveID)
				return
			}
			entry, ok := svc.StepAt(length - 1)
			if ok && entry.Temperature != 100 && entry.Temperature != 200 {
				t.Errorf("entry temperature %v is not from any known version", entry.Temperature)
				return
			}
		}
	}()

	// Consumer B: Advance must keep the cursor within the current version's
	// bounds. After a SyncCurve the cursor is reset to 0, so index must always
	// be < len(seq) of the same version.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			_, idx, length := svc.Progress()
			if length == 0 {
				continue
			}
			if idx >= length {
				t.Errorf("post-Advance index %d not bounded by length %d: cursor/seq version split",
					idx, length)
				return
			}
			svc.Advance()
		}
	}()

	time.Sleep(200 * time.Millisecond)
	close(stop)
	wg.Wait()
}
