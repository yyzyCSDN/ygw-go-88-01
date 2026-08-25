package esd

import "chemicalprocessdcs/internal/model"

func (s *Service) MarkRestored(id string) error {
	return s.writebackRestore(id)
}

func (s *Service) writebackRestore(id string) error {
	if s.valve(id, true) != nil {
		return model.ErrRestoreFailed
	}
	s.mu.Lock()
	s.states[id] = model.InterlockReset
	s.writebacks++
	s.mu.Unlock()
	return nil
}

func (s *Service) Reset(id string) {
	s.mu.Lock()
	s.states[id] = model.InterlockFree
	s.mu.Unlock()
}
