package esd_test

import (
	"context"
	"testing"

	"chemicalprocessdcs/internal/esd"
	"chemicalprocessdcs/internal/feed"
	"chemicalprocessdcs/internal/model"
	"chemicalprocessdcs/internal/param"
)

func TestEsdErrorNotSwallowed(t *testing.T) {
	params := param.NewStore()
	params.SetParams("main", model.ProcessParams{TargetTemperature: 100, TargetPressure: 5, FeedTarget: 10, CoolTarget: 50, CurveID: "c"})
	params.SetCurve(model.Curve{ID: "c", Steps: []model.CurveStep{{Index: 0, Temperature: 100, Pressure: 5}}})
	confirm := func(ctx context.Context, id string) (model.FeedConfirmation, error) {
		return model.FeedConfirmation{Amount: 1, OK: true}, nil
	}
	feedSvc := feed.NewService(params, confirm)
	valve := func(id string, open bool) error { return model.ErrValveFailed }
	esdSvc := esd.NewService(feedSvc, valve)

	err := esdSvc.Trip("R1")
	if err == nil {
		t.Fatal("trip must return the valve error")
	}
	if esdSvc.State("R1") == model.InterlockTripped {
		t.Fatal("failed trip must not mark reactor tripped")
	}
}
