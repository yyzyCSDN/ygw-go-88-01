package reactor

import (
	"time"

	"chemicalprocessdcs/internal/alarm"
	"chemicalprocessdcs/internal/model"
)

func (s *Service) Step(id string) error {
	r, err := s.Get(id)
	if err != nil {
		return err
	}
	params, ok := s.mainParams()
	if !ok {
		return model.ErrParamNotFound
	}

	if err := s.esd.CheckInterlock(r, params); err != nil {
		return err
	}
	_ = s.alarm.ApplyRules(alarm.DefaultThresholdRules(), id, r.Temperature, r.Pressure)
	_, _ = s.PressureDeviation(id, params.TargetPressure)

	s.feed.SyncCurve()
	s.cool.SyncCurve()

	switch r.State {
	case model.ReactorIdle:
		if err := s.beginHeating(id); err != nil {
			return err
		}
	case model.ReactorHeating:
		if r.Temperature >= params.TargetTemperature {
			if err := s.waitForStable(id, params.TargetTemperature); err != nil {
				s.feed.Cancel(id)
				return err
			}
			if err := s.beginReacting(id); err != nil {
				return err
			}
		}
	case model.ReactorReacting:
		if err := s.driveFeeding(id, params); err != nil {
			return err
		}
		if s.reactionDone(id, params) {
			_ = s.cool.RampTo(params.CoolTarget, 3)
			if err := s.beginCooling(id); err != nil {
				return err
			}
		}
	case model.ReactorCooling:
		if r.Temperature <= params.CoolTarget {
			_ = s.cool.Control(r.Temperature, params.CoolTarget)
			if err := s.finishCooling(id); err != nil {
				return err
			}
		}
	}
	return s.recordRun(id)
}

func (s *Service) beginHeating(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.reactors[id]
	from := r.State
	r.State = model.ReactorHeating
	s.recordTransitionLocked(id, from, model.ReactorHeating)
	return nil
}

func (s *Service) beginReacting(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.reactors[id]
	from := r.State
	r.State = model.ReactorReacting
	s.recordTransitionLocked(id, from, model.ReactorReacting)
	return nil
}

func (s *Service) beginCooling(id string) error {
	s.feed.Stop(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.reactors[id]
	from := r.State
	r.State = model.ReactorCooling
	s.recordTransitionLocked(id, from, model.ReactorCooling)
	return s.cool.Cool(id, 0)
}

func (s *Service) finishCooling(id string) error {
	if err := s.cool.Stop(id); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.reactors[id]
	from := r.State
	r.State = model.ReactorIdle
	r.Tripped = false
	s.recordTransitionLocked(id, from, model.ReactorIdle)
	return nil
}

func (s *Service) driveFeeding(id string, params model.ProcessParams) error {
	if s.feed.State(id) == model.FeedIdle {
		if err := s.feed.Start(id); err != nil {
			return err
		}
	}
	if s.feed.State(id) == model.FeedMetering {
		if err := s.feed.MeterWithRetry(id, 2); err != nil {
			s.feed.Reset(id)
			return err
		}
	}
	s.feed.Advance()
	_ = params
	return nil
}

func (s *Service) reactionDone(id string, params model.ProcessParams) bool {
	_ = params
	return s.feed.BatchComplete(id)
}

func (s *Service) waitForStable(id string, target float64) error {
	deadline := time.Now().Add(time.Duration(s.cfg.StableTimeout) * time.Second)
	for time.Now().Before(deadline) {
		temperature, err := s.Temperature(id)
		if err != nil {
			return err
		}
		diff := temperature - target
		if diff < 0 {
			diff = -diff
		}
		if diff <= s.cfg.StableTolerance {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return model.ErrReactorBusy
}

func (s *Service) recordRun(id string) error {
	r, err := s.Get(id)
	if err != nil {
		return err
	}
	return s.rec.Append(model.RecordEntry{
		ReactorID:   r.ID,
		State:       r.State,
		Temperature: r.Temperature,
		Pressure:    r.Pressure,
	})
}
