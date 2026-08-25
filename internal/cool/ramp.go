package cool

func CoolingRampTarget(current float64, target float64, step int) float64 {
	if step <= 0 {
		return current
	}
	delta := (current - target) / float64(step)
	if delta < 0 {
		return target
	}
	return current - delta
}

func (s *Service) RampTo(target float64, steps int) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := 0.0
	if len(s.seq) > 0 {
		current = s.seq[len(s.seq)-1].Temperature
	}
	return CoolingRampTarget(current, target, steps)
}
