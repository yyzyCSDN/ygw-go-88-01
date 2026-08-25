package reactor

import "chemicalprocessdcs/internal/model"

type Snapshot struct {
	Reactors map[string]model.Reactor
}

func (s *Service) TakeSnapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	reactors := make(map[string]model.Reactor, len(s.reactors))
	for id, r := range s.reactors {
		reactors[id] = *r
	}
	return Snapshot{Reactors: reactors}
}

func (s *Service) Recover() {
	s.applyRecover(s.cachedRestored)
}

func (s *Service) applyRecover(restored map[string]bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, r := range s.reactors {
		if restored[id] {
			r.Restored = true
			r.Tripped = false
			continue
		}
		r.Restored = false
		r.Tripped = true
	}
}
