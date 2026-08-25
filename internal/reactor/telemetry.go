package reactor

type Telemetry struct {
	Statuses []Status
	Count    int
}

func (s *Service) Telemetry() Telemetry {
	statuses := s.AllStatuses()
	return Telemetry{Statuses: statuses, Count: len(statuses)}
}

func (t Telemetry) ReactorCount() int {
	return t.Count
}

func (t Telemetry) TrippedIDs() []string {
	ids := make([]string, 0)
	for _, status := range t.Statuses {
		if status.Tripped {
			ids = append(ids, status.ID)
		}
	}
	return ids
}

