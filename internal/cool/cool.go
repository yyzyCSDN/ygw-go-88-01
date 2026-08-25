package cool

import (
	"sync"

	"chemicalprocessdcs/internal/model"
	"chemicalprocessdcs/internal/param"
)

type Service struct {
	mu       sync.Mutex
	active   map[string]bool
	curveGen uint64
	seq      []param.SequenceEntry
	version  uint64
	store    *param.Store
	valve    func(id string, open bool) error
}

func NewService(store *param.Store, valve func(id string, open bool) error) *Service {
	return &Service{
		active: make(map[string]bool),
		store:  store,
		valve:  valve,
	}
}

func (s *Service) SyncCurve() {
	gen := s.store.Generation()
	if gen == s.curveGen {
		return
	}
	s.curveGen = gen
	params, ok := s.store.Params("main")
	if !ok {
		s.seq = nil
		return
	}
	curve, ok := s.store.Curve(params.CurveID)
	if !ok {
		s.seq = nil
		return
	}
	s.seq = param.BuildSequence(curve)
}

func (s *Service) Version() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.version
}

func (s *Service) Cool(id string, stepIndex int) error {
	entry, ok := s.StepAt(stepIndex)
	if !ok {
		return model.ErrCurveNotFound
	}
	if err := s.valve(id, true); err != nil {
		return err
	}
	zones := s.ZonesFor(entry.Temperature, entry.Temperature*0.5, 2)
	_ = zones
	s.mu.Lock()
	s.active[id] = true
	s.mu.Unlock()
	_ = entry
	return nil
}

func (s *Service) Stop(id string) error {
	return s.stopWithRetry(id, 3)
}

func (s *Service) stopWithRetry(id string, attempts int) error {
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if err := s.valve(id, false); err != nil {
			lastErr = err
			continue
		}
		s.mu.Lock()
		s.active[id] = false
		s.mu.Unlock()
		return nil
	}
	return lastErr
}

func (s *Service) Active(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active[id]
}

func (s *Service) StepAt(index int) (param.SequenceEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return param.CurrentStep(s.seq, index)
}

func (s *Service) SequenceLength() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.seq)
}
