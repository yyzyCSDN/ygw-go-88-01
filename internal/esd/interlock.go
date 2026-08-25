package esd

func (s *Service) InterlockedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, state := range s.states {
		if state == "interlocked" || state == "tripped" {
			count++
		}
	}
	return count
}
