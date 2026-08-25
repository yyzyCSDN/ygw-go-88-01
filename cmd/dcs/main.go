package main

import (
	"context"
	"log"
	"path/filepath"

	"chemicalprocessdcs/internal/alarm"
	"chemicalprocessdcs/internal/cool"
	"chemicalprocessdcs/internal/esd"
	"chemicalprocessdcs/internal/feed"
	"chemicalprocessdcs/internal/model"
	"chemicalprocessdcs/internal/param"
	"chemicalprocessdcs/internal/reactor"
	"chemicalprocessdcs/internal/record"
)

func main() {
	cfg := LoadConfig()

	params := param.NewStore()
	params.SetParams("main", model.ProcessParams{
		TargetTemperature: 180,
		TargetPressure:    6.0,
		FeedTarget:        40,
		CoolTarget:        80,
		CurveID:           "default",
	})
	params.SetCurve(model.Curve{
		ID: "default",
		Steps: []model.CurveStep{
			{Index: 0, Temperature: 120, Pressure: 4.0, HoldSeconds: 60},
			{Index: 1, Temperature: 180, Pressure: 6.0, HoldSeconds: 120},
			{Index: 2, Temperature: 80, Pressure: 2.0, HoldSeconds: 60},
		},
	})
	paramsFile := filepath.Join(cfg.Dir, "params.json")
	if err := params.Load(paramsFile); err != nil {
		_ = params.Save(paramsFile)
	}

	valve := func(id string, open bool) error { return nil }
	confirm := func(ctx context.Context, id string) (model.FeedConfirmation, error) {
		return model.FeedConfirmation{Amount: 1, OK: true}, nil
	}

	feedSvc := feed.NewService(params, confirm)
	feedSvc.AddCascade(feed.Cascade{ID: "C1", Enabled: true, Capacity: 20})
	feedSvc.AddCascade(feed.Cascade{ID: "C2", Enabled: true, Capacity: 20})
	coolSvc := cool.NewService(params, valve)
	esdSvc := esd.NewService(feedSvc, valve)
	alarmSvc := alarm.NewService(esdSvc)
	recSvc := record.NewService(cfg.Dir)
	journal := record.NewJournal(cfg.Dir)
	reactorSvc := reactor.NewService(params, coolSvc, feedSvc, esdSvc, alarmSvc, recSvc)
	stateFile := filepath.Join(cfg.Dir, "reactor-state.json")
	_ = reactorSvc.LoadState(stateFile)
	defer reactorSvc.SaveState(stateFile)

	server := NewServer(cfg, reactorSvc, esdSvc, alarmSvc, params, journal, recSvc)
	log.Printf("dcs control listening on %s", cfg.Addr)
	if err := server.Start(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
