package alarm_test

import (
	"context"
	"testing"

	"chemicalprocessdcs/internal/alarm"
	"chemicalprocessdcs/internal/esd"
	"chemicalprocessdcs/internal/feed"
	"chemicalprocessdcs/internal/model"
	"chemicalprocessdcs/internal/param"
)

func TestRestoreWritebackErrorNotSwallowed(t *testing.T) {
	params := param.NewStore()
	params.SetParams("main", model.ProcessParams{FeedTarget: 10})
	confirm := func(ctx context.Context, id string) (model.FeedConfirmation, error) {
		return model.FeedConfirmation{Amount: 1, OK: true}, nil
	}
	feedSvc := feed.NewService(params, confirm)
	valve := func(id string, open bool) error { return model.ErrValveFailed }
	esdSvc := esd.NewService(feedSvc, valve)
	alarmSvc := alarm.NewService(esdSvc)

	if err := alarmSvc.Restore("R1"); err == nil {
		t.Fatal("restore writeback error must not be swallowed")
	}
}
