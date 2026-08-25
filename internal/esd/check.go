package esd

import "chemicalprocessdcs/internal/model"

func (s *Service) CheckInterlock(r *model.Reactor, params model.ProcessParams) error {
	if r.Temperature > s.maxTemp || r.Pressure > s.maxPressure {
		s.reasons[r.ID] = s.reasonFor(r, s.maxTemp, s.maxPressure)
		return s.Trip(r.ID)
	}
	if r.Pressure > params.TargetPressure && r.State == model.ReactorReacting {
		return model.ErrValveFailed
	}
	return nil
}
