package feed

import (
	"context"
	"testing"
	"time"

	"chemicalprocessdcs/internal/model"
	"chemicalprocessdcs/internal/param"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	store := param.NewStore()
	store.SetParams("main", model.ProcessParams{
		TargetTemperature: 180,
		TargetPressure:    6.0,
		FeedTarget:        40,
		CoolTarget:        80,
		CurveID:           "default",
	})
	confirm := func(ctx context.Context, id string) (model.FeedConfirmation, error) {
		return model.FeedConfirmation{Amount: 1, OK: true}, nil
	}
	return NewService(store, confirm)
}

// TestBatchDoesNotDeadlock proves the reentrant-lock fix: Batch/BatchComplete
// hold s.mu and must not call back into a method that re-locks it. Before the
// fix this self-deadlocked (sync.Mutex is not reentrant) on every call.
func TestBatchDoesNotDeadlock(t *testing.T) {
	s := newTestService(t)
	s.Register("R1")

	done := make(chan struct{})
	go func() {
		b := s.Batch("R1")
		if b.TargetAmount != 40 {
			t.Errorf("TargetAmount = %v, want 40", b.TargetAmount)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Batch deadlocked (reentrant mutex on s.mu)")
	}

	if s.BatchComplete("R1") {
		t.Errorf("BatchComplete = true, want false before any feed")
	}
}
