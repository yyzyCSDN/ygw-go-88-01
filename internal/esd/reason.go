package esd

import "chemicalprocessdcs/internal/model"

type TripReason string

const (
	TripReasonOverTemperature TripReason = "over_temperature"
	TripReasonOverPressure    TripReason = "over_pressure"
	TripReasonManual          TripReason = "manual"
)

func (s *Service) TripReason(id string) TripReason {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reasons[id]
}

func (s *Service) reasonFor(r *model.Reactor, maxTemp, maxPressure float64) TripReason {
	if r.Temperature > maxTemp {
		return TripReasonOverTemperature
	}
	if r.Pressure > maxPressure {
		return TripReasonOverPressure
	}
	return TripReasonManual
}
