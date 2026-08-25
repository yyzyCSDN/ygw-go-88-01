package param

import (
	"sync"
	"testing"
	"time"

	"chemicalprocessdcs/internal/model"
)

// makeCurve builds a curve whose every step carries a single distinguishing
// temperature so a mismatched params/curve pair can be detected.
func makeCurve(id string, temp float64) model.Curve {
	return model.Curve{
		ID: id,
		Steps: []model.CurveStep{
			{Index: 0, Temperature: temp, Pressure: 1.0, HoldSeconds: 60},
			{Index: 1, Temperature: temp, Pressure: 1.0, HoldSeconds: 60},
		},
	}
}

// TestCurveSnapshotAtomicity hammers CurveSnapshot (reader) against a producer
// that concurrently switches the active curve: it alternates SetParams so the
// "main" params reference curve "a" (temperature 100) vs curve "b" (temperature
// 200), and sets both curves. A non-atomic snapshot (params read under one
// lock, curve under another) could return params pointing at curve "a" while
// the curve field was populated from a stale or just-switched "b", i.e. a
// params/curve pair that belong to two different store versions.
//
// The invariant under the fix: when CurveSnapshot reports OK, the returned
// Curve's ID must equal the returned Params' CurveID, and every step
// temperature must match the version that CurveID implies. If a snapshot ever
// returns a curve whose ID disagrees with the params' CurveID, that is the
// half-old/half-new bug and the test fails.
func TestCurveSnapshotAtomicity(t *testing.T) {
	store := NewStore()
	// Two distinct curves, each internally consistent at one temperature.
	store.SetCurve(makeCurve("a", 100))
	store.SetCurve(makeCurve("b", 200))
	store.SetParams("main", model.ProcessParams{
		TargetTemperature: 180,
		CurveID:           "a",
	})

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Producer: flip the active CurveID between "a" and "b" and refresh the
	// curve contents, bumping the store generation each time.
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
			id := "a"
			temp := 100.0
			if i%2 == 1 {
				id = "b"
				temp = 200.0
			}
			store.SetCurve(makeCurve(id, temp))
			store.SetParams("main", model.ProcessParams{
				TargetTemperature: 180,
				CurveID:           id,
			})
			i++
		}
	}()

	// Readers: every snapshot must be internally consistent — the curve's ID
	// must equal the params' CurveID, and every step temperature must match
	// that curve's signature temperature.
	readers := 8
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
				snap := store.CurveSnapshot("main", 0)
				if !snap.OK {
					continue
				}
				if snap.Curve.ID != snap.Params.CurveID {
					t.Errorf("snapshot straddles a switch: params.CurveID=%q but curve.ID=%q",
						snap.Params.CurveID, snap.Curve.ID)
					return
				}
				want := 100.0
				if snap.Curve.ID == "b" {
					want = 200.0
				}
				for _, step := range snap.Curve.Steps {
					if step.Temperature != want {
						t.Errorf("curve %q has step temperature %v, want %v (mixed versions)",
							snap.Curve.ID, step.Temperature, want)
						return
					}
				}
			}
		}()
	}

	time.Sleep(300 * time.Millisecond)
	close(stop)
	wg.Wait()
}
