package feed

import "chemicalprocessdcs/internal/model"

func (s *Service) MeterWithRetry(id string, attempts int) error {
	if attempts < 1 {
		attempts = 1
	}
	var last error
	for i := 0; i < attempts; i++ {
		if s.State(id) != model.FeedMetering {
			if err := s.Start(id); err != nil {
				return err
			}
		}
		if err := s.Meter(id); err == nil {
			return nil
		} else {
			last = err
		}
	}
	return last
}
