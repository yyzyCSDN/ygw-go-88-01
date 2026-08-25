package cool

func (s *Service) Control(current float64, target float64) float64 {
	if current <= target {
		return 0
	}
	delta := current - target
	if delta > 1 {
		return 1
	}
	return delta
}
