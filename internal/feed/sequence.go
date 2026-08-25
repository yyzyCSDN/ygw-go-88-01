package feed

func (s *Service) CurrentIndex() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncCurveLocked()
	return s.index
}

// Advance moves the feed cursor to the next step. It reconciles against the
// store generation first, so index and the sequence length it is bounded by
// always belong to the same version — the step it advances past is the same
// step it was pointing at.
func (s *Service) Advance() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncCurveLocked()
	if s.index < len(s.seq)-1 {
		s.index++
	}
	return s.index
}
