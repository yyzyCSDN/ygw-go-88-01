package feed

func (s *Service) CurrentIndex() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.index
}

func (s *Service) Advance() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index < len(s.seq)-1 {
		s.index++
	}
	return s.index
}

