package feed

type Batch struct {
	TargetAmount float64
	Delivered    float64
}

func (s *Service) Batch(id string) Batch {
	s.mu.Lock()
	defer s.mu.Unlock()
	target := s.feedTargetLocked()
	return Batch{TargetAmount: target, Delivered: s.amount[id]}
}

func (s *Service) BatchComplete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	target := s.feedTargetLocked()
	return target > 0 && s.amount[id] >= target
}

