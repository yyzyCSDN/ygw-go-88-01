package cool

type Zone struct {
	ID     string
	Target float64
	Open   bool
}

func (s *Service) ZonesFor(current float64, target float64, count int) []Zone {
	if count <= 0 {
		return nil
	}
	zones := make([]Zone, 0, count)
	step := (current - target) / float64(count)
	for i := 0; i < count; i++ {
		zones = append(zones, Zone{
			ID:     "Z" + string(rune('A'+i)),
			Target: current - step*float64(i+1),
			Open:   true,
		})
	}
	return zones
}
