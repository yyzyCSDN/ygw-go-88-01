package esd

import (
	"context"
	"errors"
	"sync"
	"testing"

	"chemicalprocessdcs/internal/feed"
	"chemicalprocessdcs/internal/model"
	"chemicalprocessdcs/internal/param"
)

// newTestService 构造一个可注入阀门函数的 ESD 服务。
func newTestService(valve func(id string, open bool) error) *Service {
	store := param.NewStore()
	confirm := func(context.Context, string) (model.FeedConfirmation, error) {
		return model.FeedConfirmation{Amount: 1, OK: true}, nil
	}
	feedSvc := feed.NewService(store, confirm)
	return &Service{
		states:      make(map[string]model.InterlockState),
		feed:        feedSvc,
		valve:       valve,
		maxTemp:     model.DefaultControlConfig().MaxTemperature,
		maxPressure: model.DefaultControlConfig().MaxPressure,
		reasons:     make(map[string]TripReason),
	}
}

// Trip 在进料阀关闭失败时，绝不应置为已停车，且必须返回错误以便上层重试。
func TestTrip_ValveFailureNotMarkedTripped(t *testing.T) {
	var valveCalls int
	s := newTestService(func(id string, open bool) error {
		valveCalls++
		return errors.New("actuator stuck")
	})

	err := s.Trip("R1")
	if err == nil {
		t.Fatal("valve failure should return an error, got nil")
	}
	if !errors.Is(err, model.ErrValveFailed) {
		t.Fatalf("expected ErrValveFailed, got %v", err)
	}
	if got := s.State("R1"); got == model.InterlockTripped {
		t.Fatalf("reactor must NOT be marked tripped when the valve did not move, got %q", got)
	}
	if valveCalls != 1 {
		t.Fatalf("valve should be exercised once, got %d calls", valveCalls)
	}
}

// Trip 在阀门真正关闭后应置为已停车并停止进料。
func TestTrip_ValveSuccessMarksTripped(t *testing.T) {
	var closed bool
	s := newTestService(func(id string, open bool) error {
		closed = !open
		return nil
	})
	s.feed.Register("R1")

	if err := s.Trip("R1"); err != nil {
		t.Fatalf("unexpected error on successful trip: %v", err)
	}
	if got := s.State("R1"); got != model.InterlockTripped {
		t.Fatalf("expected tripped after successful valve close, got %q", got)
	}
	if !closed {
		t.Fatal("feed valve was not closed on trip")
	}
	if got := s.feed.State("R1"); got != model.FeedStopped {
		t.Fatalf("expected feed stopped after trip, got %q", got)
	}
	if got := s.TripReason("R1"); got != TripReasonManual {
		t.Fatalf("expected manual trip reason, got %q", got)
	}
}

// 阀门失败后重试，成功时应正常停车——验证可重试。
func TestTrip_RetrySucceedsAfterFailure(t *testing.T) {
	var attempts int
	s := newTestService(func(id string, open bool) error {
		attempts++
		if attempts < 2 {
			return errors.New("transient")
		}
		return nil
	})

	if err := s.Trip("R1"); err == nil {
		t.Fatal("first attempt should fail")
	}
	if got := s.State("R1"); got == model.InterlockTripped {
		t.Fatal("reactor must not be tripped after failed attempt")
	}

	if err := s.Trip("R1"); err != nil {
		t.Fatalf("retry should succeed, got %v", err)
	}
	if got := s.State("R1"); got != model.InterlockTripped {
		t.Fatalf("expected tripped after successful retry, got %q", got)
	}
}

// 避免 race detector 误报：并发 Trip 不应 panic。
func TestTrip_Concurrent(t *testing.T) {
	s := newTestService(func(id string, open bool) error { return nil })
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.Trip("R1")
		}()
	}
	wg.Wait()
	if got := s.State("R1"); got != model.InterlockTripped {
		t.Fatalf("expected tripped after concurrent trips, got %q", got)
	}
}
