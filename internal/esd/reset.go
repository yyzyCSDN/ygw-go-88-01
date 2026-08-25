package esd

import "chemicalprocessdcs/internal/model"

func (s *Service) MarkRestored(id string) error {
	_ = s.valve(id, true)
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
