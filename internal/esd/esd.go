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
	if err := s.closeValveWithRetry(id, 3); err != nil {
		return err
	}
	s.mu.Lock()
	s.states[id] = model.InterlockTripped
	s.mu.Unlock()
	s.feed.Stop(id)
	return nil
}

func (s *Service) closeValveWithRetry(id string, attempts int) error {
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if err := s.valve(id, false); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return lastErr
}

func (s *Service) State(id string) model.InterlockState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.states[id]
}

