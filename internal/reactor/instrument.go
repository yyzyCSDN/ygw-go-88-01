package reactor

import (
	"fmt"

	"chemicalprocessdcs/internal/model"
)

func ValidateReading(temperature float64, pressure float64) error {
	if temperature < -273.15 {
		return fmt.Errorf("temperature below absolute zero")
	}
	if temperature > 1200 {
		return fmt.Errorf("temperature out of instrument range")
	}
	if pressure < 0 {
		return fmt.Errorf("pressure must not be negative")
	}
	if pressure > 100 {
		return fmt.Errorf("pressure out of instrument range")
	}
	return nil
}

func (s *Service) MovingAverage(id string, window int) (float64, error) {
	if window < 1 {
		window = 1
	}
	trend, ok := s.Trend(id)
	if !ok {
		return 0, model.ErrReactorNotFound
	}
	if trend.Size() == 0 {
		return 0, nil
	}
	start := trend.Size() - window
	if start < 0 {
		start = 0
	}
	var sum float64
	count := 0
	for _, point := range trend.Points[start:] {
		sum += point.Temperature
		count++
	}
	if count == 0 {
		return 0, nil
	}
	return sum / float64(count), nil
}

func (s *Service) ReadingDelta(id string) (float64, error) {
	trend, ok := s.Trend(id)
	if !ok {
		return 0, model.ErrReactorNotFound
	}
	if trend.Size() < 2 {
		return 0, nil
	}
	last := trend.Points[len(trend.Points)-1]
	prev := trend.Points[len(trend.Points)-2]
	return last.Temperature - prev.Temperature, nil
}
