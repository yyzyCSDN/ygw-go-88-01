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
	s.refreshCachedRestored()
	s.applyRecover(s.cachedRestored)
}

func (s *Service) refreshCachedRestored() {
	s.mu.Lock()
	defer s.mu.Unlock()
	latest := s.alarm.RecoverSnapshot()
	s.cachedRestored = make(map[string]bool, len(latest))
	for id, restored := range latest {
		s.cachedRestored[id] = restored
	}
	for id := range s.reactors {
		if _, ok := s.cachedRestored[id]; !ok {
			s.cachedRestored[id] = false
		}
	}
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
