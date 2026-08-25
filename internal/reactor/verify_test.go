package reactor_test

import (
	"context"
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

func newReactor(t *testing.T) *reactor.Service {
	params := param.NewStore()
	params.SetParams("main", model.ProcessParams{TargetTemperature: 100, TargetPressure: 10, FeedTarget: 10, CoolTarget: 50, CurveID: "c"})
	params.SetCurve(model.Curve{ID: "c", Steps: []model.CurveStep{{Index: 0, Temperature: 100, Pressure: 10}}})
	confirm := func(ctx context.Context, id string) (model.FeedConfirmation, error) {
		return model.FeedConfirmation{Amount: 1, OK: true}, nil
	}
	valve := func(id string, open bool) error { return nil }
	feedSvc := feed.NewService(params, confirm)
	coolSvc := cool.NewService(params, valve)
	esdSvc := esd.NewService(feedSvc, valve)
	alarmSvc := alarm.NewService(esdSvc)
	recSvc := record.NewService(t.TempDir())
	return reactor.NewService(params, coolSvc, feedSvc, esdSvc, alarmSvc, recSvc)
}

func TestEmptyReactorNoNilPanic(t *testing.T) {
	svc := newReactor(t)
	r, err := svc.Get("missing")
	if err == nil {
		t.Fatal("missing reactor must return an error")
	}
	if r != nil {
		t.Fatal("missing reactor must return a nil value")
	}
}
