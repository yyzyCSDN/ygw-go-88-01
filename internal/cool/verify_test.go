package cool_test

import (
	"sync"
	"testing"

	"chemicalprocessdcs/internal/cool"
	"chemicalprocessdcs/internal/model"
	"chemicalprocessdcs/internal/param"
)

func TestCurveDispatchRaceFree(t *testing.T) {
	store := param.NewStore()
	store.SetParams("main", model.ProcessParams{CurveID: "c1"})
	store.SetCurve(model.Curve{ID: "c1", Steps: []model.CurveStep{{Index: 0, Temperature: 100, Pressure: 5}}})
	valve := func(id string, open bool) error { return nil }
	coolSvc := cool.NewService(store, valve)

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			coolSvc.SyncCurve()
		}()
	}
	wg.Wait()
	if coolSvc.SequenceLength() != 1 {
		t.Fatal("curve sequence must stay consistent")
	}
	if coolSvc.Version() == 0 {
		t.Fatal("curve version must be recorded")
	}
}
