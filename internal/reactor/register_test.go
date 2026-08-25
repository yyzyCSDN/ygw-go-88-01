package reactor

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"chemicalprocessdcs/internal/alarm"
	"chemicalprocessdcs/internal/cool"
	"chemicalprocessdcs/internal/esd"
	"chemicalprocessdcs/internal/feed"
	"chemicalprocessdcs/internal/model"
	"chemicalprocessdcs/internal/param"
	"chemicalprocessdcs/internal/record"
)

// newTestService builds a reactor.Service with lightweight, in-memory
// dependencies so that Register / persistence can be exercised without files.
func newTestService(t *testing.T) *Service {
	t.Helper()
	params := param.NewStore()
	params.SetParams("main", model.ProcessParams{
		TargetTemperature: 180,
		TargetPressure:    6.0,
		FeedTarget:        40,
		CoolTarget:        80,
		CurveID:           "default",
	})
	params.SetCurve(model.Curve{ID: "default", Steps: []model.CurveStep{
		{Index: 0, Temperature: 120, Pressure: 4.0, HoldSeconds: 60},
	}})

	valve := func(id string, open bool) error { return nil }
	confirm := func(ctx context.Context, id string) (model.FeedConfirmation, error) {
		return model.FeedConfirmation{Amount: 1, OK: true}, nil
	}

	feedSvc := feed.NewService(params, confirm)
	coolSvc := cool.NewService(params, valve)
	esdSvc := esd.NewService(feedSvc, valve)
	alarmSvc := alarm.NewService(esdSvc)
	recSvc := record.NewService(t.TempDir())
	return NewService(params, coolSvc, feedSvc, esdSvc, alarmSvc, recSvc)
}

// TestRegisterConcurrent reproduces the "concurrent map writes" crash that
// occurred when batch-registering reactors across goroutines: both
// reactor.Register and feed.Register touched their maps without holding the
// mutex. Run with -race to catch the data race; without the fix this test
// crashes the process.
func TestRegisterConcurrent(t *testing.T) {
	s := newTestService(t)

	const goroutines = 32
	const perG = 16
	ids := make([]string, 0, goroutines*perG)
	for g := 0; g < goroutines; g++ {
		for i := 0; i < perG; i++ {
			ids = append(ids, fmt.Sprintf("R-%03d", g*perG+i))
		}
	}

	var wg sync.WaitGroup
	// Shuffle-ish: each goroutine registers a disjoint range so the full set is
	// covered, but the work overlaps in time.
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				s.Register(ids[start*perG+i])
			}
		}(g)
	}
	wg.Wait()

	want := goroutines * perG
	if got := s.ReactorCount(); got != want {
		t.Fatalf("reactor count: got %d, want %d (registrations were lost or duplicated)", got, want)
	}
	// Every reactor must also have a matching feed registration (no missing
	// feed entries, no duplicates). feed.Register is idempotent and is the
	// source of truth for the per-reactor feed state, so its registered set
	// must equal the reactor set.
	for _, id := range ids {
		if got := s.feed.State(id); got != model.FeedIdle {
			t.Fatalf("feed state for %s: got %q, want %q (feed registration lost)", id, got, model.FeedIdle)
		}
	}
}

// TestRegisterIdempotent ensures re-registering the same reactor does not
// duplicate or reset state — the "不重复" requirement.
func TestRegisterIdempotent(t *testing.T) {
	s := newTestService(t)
	s.Register("R-1")
	s.Register("R-1") // duplicate must be a no-op
	s.Register("R-1")

	if s.ReactorCount() != 1 {
		t.Fatalf("reactor count: got %d, want 1", s.ReactorCount())
	}
	if got := s.feed.RegisteredCount(); got != 1 {
		t.Fatalf("feed registered count: got %d, want 1", got)
	}
}

// TestLoadStateRehydratesFeed verifies the restart path: after loading
// persisted state, every reactor must be re-registered with the feed service
// so that "进料登记" is not lost across restarts.
func TestLoadStateRehydratesFeed(t *testing.T) {
	s := newTestService(t)
	for _, id := range []string{"R-A", "R-B", "R-C"} {
		s.Register(id)
	}

	path := t.TempDir() + "/reactor-state.json"
	if err := s.SaveState(path); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	// Fresh service simulates a restart.
	fresh := newTestService(t)
	if err := fresh.LoadState(path); err != nil {
		t.Fatalf("LoadState: %v", err)
	}

	if got := fresh.ReactorCount(); got != 3 {
		t.Fatalf("reactor count after reload: got %d, want 3", got)
	}
	for _, id := range []string{"R-A", "R-B", "R-C"} {
		if got := fresh.feed.State(id); got != model.FeedIdle {
			t.Fatalf("feed state for %s after reload: got %q, want %q (feed registration lost)", id, got, model.FeedIdle)
		}
	}
}
