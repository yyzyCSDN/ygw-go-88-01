package main

import (
	"context"
	"errors"
	"testing"

	"chemicalprocessdcs/internal/alarm"
	"chemicalprocessdcs/internal/cool"
	"chemicalprocessdcs/internal/esd"
	"chemicalprocessdcs/internal/feed"
	"chemicalprocessdcs/internal/model"
	"chemicalprocessdcs/internal/param"
	"chemicalprocessdcs/internal/reactor"
	"chemicalprocessdcs/internal/record"
)

// newReactorSvc wires a reactor.Service mirroring main.go so we can exercise the
// interlock scan path without HTTP.
func newReactorSvc(t *testing.T) *reactor.Service {
	t.Helper()
	params := param.NewStore()
	params.SetParams("main", model.ProcessParams{
		TargetTemperature: 180, TargetPressure: 6.0,
		FeedTarget: 40, CoolTarget: 80, CurveID: "default",
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
	return reactor.NewService(params, coolSvc, feedSvc, esdSvc, alarmSvc, recSvc)
}

// TestStepUnknownReactorDoesNotPanic asserts the old nil-deref crash is gone:
// stepping an unregistered reactor returns ErrReactorNotFound instead.
func TestStepUnknownReactorDoesNotPanic(t *testing.T) {
	s := newReactorSvc(t)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("service panicked on unknown reactor: %v", r)
		}
	}()
	err := s.Step("R-999-unregistered")
	if !errors.Is(err, model.ErrReactorNotFound) {
		t.Fatalf("expected ErrReactorNotFound, got %v", err)
	}
}

// TestStepRegisteredReactorStillSteps guards against over-broad handling: a
// registered reactor must still step without a not-found error.
func TestStepRegisteredReactorStillSteps(t *testing.T) {
	s := newReactorSvc(t)
	s.Register("R-1")
	if err := s.SetReading("R-1", 120, 4.0); err != nil {
		t.Fatalf("SetReading: %v", err)
	}
	if err := s.Step("R-1"); err != nil {
		t.Fatalf("Step on registered reactor failed: %v", err)
	}
}

// TestStatusUnknownReactorReturnsNotFound confirms the read path surfaces
// not-found rather than a zero-value (nil-deref) status.
func TestStatusUnknownReactorReturnsNotFound(t *testing.T) {
	s := newReactorSvc(t)
	if _, err := s.Status("R-999-unregistered"); !errors.Is(err, model.ErrReactorNotFound) {
		t.Fatalf("expected ErrReactorNotFound, got %v", err)
	}
}
