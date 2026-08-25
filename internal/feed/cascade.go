package feed

type Cascade struct {
	ID       string
	Enabled  bool
	Capacity float64
}

func (s *Service) cascades() map[string]Cascade {
	if s.cascadeMap == nil {
		s.cascadeMap = make(map[string]Cascade)
	}
	return s.cascadeMap
}

func (s *Service) AddCascade(c Cascade) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cascades()[c.ID] = c
}

func (s *Service) CascadeCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.cascades())
}

func (s *Service) TotalCascadeCapacity() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	var total float64
	for _, c := range s.cascades() {
		if c.Enabled {
			total += c.Capacity
		}
	}
	return total
}
