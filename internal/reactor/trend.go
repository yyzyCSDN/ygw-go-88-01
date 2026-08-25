package reactor

import "chemicalprocessdcs/internal/model"

func (s *Service) Trend(id string) (model.Trend, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	trend, ok := s.trends[id]
	if !ok {
		return model.Trend{}, false
	}
	return *trend, true
}

