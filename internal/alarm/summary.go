package alarm

type Summary struct {
	Active   int
	Critical int
	Warning  int
	Info     int
}

func (s *Service) Summary() Summary {
	s.mu.Lock()
	defer s.mu.Unlock()
	summary := Summary{}
	for _, a := range s.alarms {
		if a.Active {
			summary.Active++
		}
		switch a.Level {
		case "critical":
			summary.Critical++
		case "warning":
			summary.Warning++
		default:
			summary.Info++
		}
	}
	return summary
}

func (s *Service) RestoredCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, restored := range s.snapshot {
		if restored {
			count++
		}
	}
	return count
}
