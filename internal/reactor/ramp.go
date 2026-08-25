package reactor

import "chemicalprocessdcs/internal/model"

type Ramp struct {
	StartTemperature  float64
	TargetTemperature float64
	RemainingSeconds  int
}

func (s *Service) BuildHeatingRamp(current float64, target float64) Ramp {
	seconds := s.cfg.StableTimeout
	if seconds < 1 {
		seconds = 1
	}
	return Ramp{
		StartTemperature:  current,
		TargetTemperature: target,
		RemainingSeconds:  seconds,
	}
}

func (s *Service) BuildCoolingRamp(current float64, target float64) Ramp {
	seconds := s.cfg.StableTimeout
	if seconds < 1 {
		seconds = 1
	}
	return Ramp{
		StartTemperature:  current,
		TargetTemperature: target,
		RemainingSeconds:  seconds,
	}
}

func (r Ramp) NextTemperature(elapsedSeconds int) float64 {
	if elapsedSeconds <= 0 {
		return r.StartTemperature
	}
	if r.RemainingSeconds <= 0 {
		return r.TargetTemperature
	}
	if elapsedSeconds >= r.RemainingSeconds {
		return r.TargetTemperature
	}
	delta := r.TargetTemperature - r.StartTemperature
	return r.StartTemperature + delta*float64(elapsedSeconds)/float64(r.RemainingSeconds)
}

func (r Ramp) Done(elapsedSeconds int) bool {
	if r.RemainingSeconds <= 0 {
		return true
	}
	return elapsedSeconds >= r.RemainingSeconds
}

func (s *Service) RampForState(id string) (Ramp, error) {
	r, err := s.Get(id)
	if err != nil {
		return Ramp{}, err
	}
	params, ok := s.mainParams()
	if !ok {
		return Ramp{}, model.ErrParamNotFound
	}
	switch r.State {
	case model.ReactorHeating:
		return s.BuildHeatingRamp(r.Temperature, params.TargetTemperature), nil
	case model.ReactorCooling:
		return s.BuildCoolingRamp(r.Temperature, params.CoolTarget), nil
	default:
		return Ramp{}, nil
	}
}

func (s *Service) RampProgress(id string, elapsedSeconds int) (float64, bool, error) {
	ramp, err := s.RampForState(id)
	if err != nil {
		return 0, false, err
	}
	if ramp.RemainingSeconds == 0 {
		return 0, false, nil
	}
	return ramp.NextTemperature(elapsedSeconds), ramp.Done(elapsedSeconds), nil
}
