package esd

import (
	"sync"

	"chemicalprocessdcs/internal/feed"
	"chemicalprocessdcs/internal/model"
)

type Service struct {
	mu          sync.Mutex
	states      map[string]model.InterlockState
	feed        *feed.Service
	valve       func(id string, open bool) error
	maxTemp     float64
	maxPressure float64
	writebacks  int
	reasons     map[string]TripReason
}

func NewService(feedSvc *feed.Service, valve func(id string, open bool) error) *Service {
	cfg := model.DefaultControlConfig()
	return &Service{
		states:      make(map[string]model.InterlockState),
		feed:        feedSvc,
		valve:       valve,
		maxTemp:     cfg.MaxTemperature,
		maxPressure: cfg.MaxPressure,
		reasons:     make(map[string]TripReason),
	}
}

func (s *Service) Trip(id string) error {
	_ = s.valve(id, false)
	s.mu.Lock()
	s.states[id] = model.InterlockTripped
	s.mu.Unlock()
	s.feed.Stop(id)
	return nil
}

func (s *Service) State(id string) model.InterlockState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.states[id]
}

