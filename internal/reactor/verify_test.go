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

func TestReactorWaitTimeoutHandled(t *testing.T) {
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
	svc := reactor.NewService(params, coolSvc, feedSvc, esdSvc, alarmSvc, recSvc)

	svc.Register("R1")
	svc.SetConfig(model.ControlConfig{StableTolerance: 1.5, StableTimeout: 1, FeedTimeout: 8, MaxPressure: 12, MaxTemperature: 280})
	if err := svc.SetReading("R1", 110, 5); err != nil {
		t.Fatal(err)
	}
	if err := svc.Step("R1"); err != nil {
		t.Fatal(err)
	}
	if err := svc.Step("R1"); err == nil {
		t.Fatal("temperature wait timeout must be reported")
	}
}
