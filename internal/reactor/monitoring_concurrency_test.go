package reactor

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"chemicalprocessdcs/internal/alarm"
	"chemicalprocessdcs/internal/cool"
	"chemicalprocessdcs/internal/esd"
	"chemicalprocessdcs/internal/feed"
	"chemicalprocessdcs/internal/model"
	"chemicalprocessdcs/internal/param"
	"chemicalprocessdcs/internal/record"
)

func newMonitoringFixture(t *testing.T) (*Service, *record.Service) {
	t.Helper()
	dir := t.TempDir()
	store := param.NewStore()
	store.SetParams("main", model.ProcessParams{
		TargetTemperature: 180,
		TargetPressure:    6.0,
		FeedTarget:        40,
		CoolTarget:        80,
		CurveID:           "default",
	})
	store.SetCurve(model.Curve{ID: "default", Steps: []model.CurveStep{
		{Index: 0, Temperature: 120, Pressure: 4.0, HoldSeconds: 60},
		{Index: 1, Temperature: 180, Pressure: 6.0, HoldSeconds: 120},
	}})
	confirm := func(ctx context.Context, id string) (model.FeedConfirmation, error) {
		return model.FeedConfirmation{Amount: 1, OK: true}, nil
	}
	feedSvc := feed.NewService(store, confirm)
	coolSvc := cool.NewService(store, func(id string, open bool) error { return nil })
	esdSvc := esd.NewService(feedSvc, func(id string, open bool) error { return nil })
	alarmSvc := alarm.NewService(esdSvc)
	recSvc := record.NewService(dir)
	svc := NewService(store, coolSvc, feedSvc, esdSvc, alarmSvc, recSvc)
	return svc, recSvc
}

// TestConcurrentMonitoringDoesNotDropRecords reproduces the reported symptom:
// many reactor monitoring goroutines run the control loop concurrently. Before
// the feed deadlock fix, any reactor reaching ReactorReacting deadlocked in
// reactionDone -> BatchComplete and never reached recordRun, so run-log records
// were dropped and the sequence drifted. After the fix every Step appends a
// record and the persisted sequence is contiguous with no gaps.
func TestConcurrentMonitoringDoesNotDropRecords(t *testing.T) {
	svc, recSvc := newMonitoringFixture(t)

	const reactors = 8
	const steps = 25
	var wg sync.WaitGroup
	wg.Add(reactors)
	for r := 0; r < reactors; r++ {
		go func(id int) {
			defer wg.Done()
			rid := "R" + strconv.Itoa(id)
			svc.Register(rid)
			_ = svc.SetReading(rid, 120+float64(id), 5.0)
			for i := 0; i < steps; i++ {
				_ = svc.Step(rid)
				time.Sleep(time.Millisecond)
			}
		}(r)
	}

	// Fail the test (rather than hang the suite) if a goroutine deadlocked.
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("monitoring goroutines deadlocked; run-log records being dropped")
	}

	entries, err := recSvc.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no run-log records written (records dropped)")
	}
	// The persisted high-water mark must equal the number of durable lines.
	if got := recSvc.Seq(); uint64(len(entries)) != got {
		t.Errorf("Seq = %d, want %d (counter drift vs durable lines)", got, len(entries))
	}
}
