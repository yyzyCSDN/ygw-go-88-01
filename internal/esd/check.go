package esd

import "chemicalprocessdcs/internal/model"

func (s *Service) CheckInterlock(r *model.Reactor, params model.ProcessParams) error {
	return s.validateReactor(r, params)
}

func (s *Service) validateReactor(r *model.Reactor, params model.ProcessParams) error {
	if r == nil {
		return model.ErrReactorNotFound
	}
	if r.Temperature > s.maxTemp || r.Pressure > s.maxPressure {
		reason := s.reasonFor(r, s.maxTemp, s.maxPressure)
		s.mu.Lock()
		s.reasons[r.ID] = reason
		s.mu.Unlock()
		return s.Trip(r.ID)
	}
	if r.Pressure > params.TargetPressure && r.State == model.ReactorReacting {
		return model.ErrValveFailed
	}
	return nil
}
