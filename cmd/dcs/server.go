package main

import (
	"net/http"
	"os"
	"path/filepath"

	"chemicalprocessdcs/internal/alarm"
	"chemicalprocessdcs/internal/esd"
	"chemicalprocessdcs/internal/param"
	"chemicalprocessdcs/internal/reactor"
	"chemicalprocessdcs/internal/record"
)

type Server struct {
	cfg     Config
	reactor *reactor.Service
	esd     *esd.Service
	alarm   *alarm.Service
	params  *param.Store
	journal *record.Journal
	record  *record.Service
}

func NewServer(
	cfg Config,
	reactorSvc *reactor.Service,
	esdSvc *esd.Service,
	alarmSvc *alarm.Service,
	params *param.Store,
	journal *record.Journal,
	recordSvc *record.Service,
) *Server {
	return &Server{
		cfg:     cfg,
		reactor: reactorSvc,
		esd:     esdSvc,
		alarm:   alarmSvc,
		params:  params,
		journal: journal,
		record:  recordSvc,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	s.routes(mux)
	_ = os.MkdirAll(s.cfg.Dir, 0755)
	return http.ListenAndServe(s.cfg.Addr, mux)
}

func (s *Server) routes(mux *http.ServeMux) {
	console := filepath.Join("web", "console.html")
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, console)
	})
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/api/v1/reactors", s.handleListReactors)
	mux.HandleFunc("/api/v1/reactors/", s.handleReactor)
	mux.HandleFunc("/api/v1/params", s.handleParams)
	mux.HandleFunc("/api/v1/curves", s.handleCurves)
	mux.HandleFunc("/api/v1/recipes", s.handleRecipes)
	mux.HandleFunc("/api/v1/alarms", s.handleAlarms)
	mux.HandleFunc("/api/v1/records", s.handleRecords)
	mux.HandleFunc("/api/v1/journal", s.handleJournal)
	mux.HandleFunc("/api/v1/status", s.handleStatus)
	mux.HandleFunc("/api/v1/esd/trip", s.handleTrip)
	mux.HandleFunc("/api/v1/alarms/restore", s.handleRestore)
}
