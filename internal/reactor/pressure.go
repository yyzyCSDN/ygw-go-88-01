package reactor

func (s *Service) PressureDeviation(id string, target float64) (float64, error) {
	r, err := s.Get(id)
	if err != nil {
		return 0, err
	}
	return r.Pressure - target, nil
}
