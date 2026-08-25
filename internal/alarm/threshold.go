package alarm

import (
	"time"

	"chemicalprocessdcs/internal/model"
)

type ThresholdRules struct {
	MaxTemperature float64
	MaxPressure    float64
	WarnRatio      float64
}

func DefaultThresholdRules() ThresholdRules {
	return ThresholdRules{MaxTemperature: 280, MaxPressure: 12, WarnRatio: 0.9}
}

func (r ThresholdRules) Evaluate(id string, temperature float64, pressure float64) string {
	if temperature > r.MaxTemperature || pressure > r.MaxPressure {
		return "critical"
	}
	if temperature > r.MaxTemperature*r.WarnRatio || pressure > r.MaxPressure*r.WarnRatio {
		return "warning"
	}
	return "info"
}

func (s *Service) ApplyRules(rules ThresholdRules, id string, temperature float64, pressure float64) string {
	level := rules.Evaluate(id, temperature, pressure)
	s.mu.Lock()
	s.alarms[id] = model.Alarm{ReactorID: id, Level: level, Message: "threshold evaluated", Active: level != "info"}
	s.appendHistoryLocked(HistoryEntry{ReactorID: id, Level: level, Message: "threshold evaluated", At: time.Now().UTC()})
	s.mu.Unlock()
	return level
}
