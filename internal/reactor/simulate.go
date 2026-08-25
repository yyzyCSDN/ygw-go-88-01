package reactor

func (s *Service) Simulate(id string, steps int) ([]Status, error) {
	if steps <= 0 {
		steps = 1
	}
	statuses := make([]Status, 0, steps)
	for i := 0; i < steps; i++ {
		if err := s.Step(id); err != nil {
			return statuses, err
		}
		status, err := s.Status(id)
		if err != nil {
			return statuses, err
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func (s *Service) BatchReport(id string) (map[string]any, error) {
	status, err := s.Status(id)
	if err != nil {
		return nil, err
	}
	stats, _ := s.Statistics(id)
	movingAverage, _ := s.MovingAverage(id, 5)
	readingDelta, _ := s.ReadingDelta(id)
	lastTransition, _ := s.LastTransition(id)
	return map[string]any{
		"status":           status,
		"statistics":       stats,
		"moving_average":   movingAverage,
		"reading_delta":    readingDelta,
		"last_transition":  lastTransition,
		"feed_batch":       s.feed.Batch(id),
		"feed_step":        s.feed.CurrentIndex(),
		"cascade_count":    s.feed.CascadeCount(),
		"cascade_capacity": s.feed.TotalCascadeCapacity(),
		"confirmations":    s.feed.ConfirmationCount(),
		"transitions":      s.TransitionCount(id),
	}, nil
}
