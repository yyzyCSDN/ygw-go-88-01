package reactor

import "chemicalprocessdcs/internal/model"

type Statistics struct {
	AverageTemperature float64
	PeakTemperature    float64
	PeakPressure       float64
	SampleCount        int
}

func (s *Service) Statistics(id string) (Statistics, error) {
	trend, ok := s.Trend(id)
	if !ok {
		return Statistics{}, model.ErrReactorNotFound
	}
	stats := Statistics{SampleCount: trend.Size()}
	if trend.Size() == 0 {
		return stats, nil
	}
	var tempSum float64
	for _, point := range trend.Points {
		tempSum += point.Temperature
		if point.Temperature > stats.PeakTemperature {
			stats.PeakTemperature = point.Temperature
		}
		if point.Pressure > stats.PeakPressure {
			stats.PeakPressure = point.Pressure
		}
	}
	stats.AverageTemperature = tempSum / float64(trend.Size())
	return stats, nil
}
