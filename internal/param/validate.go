package param

import (
	"fmt"

	"chemicalprocessdcs/internal/model"
)

func ValidateParams(p model.ProcessParams) error {
	if p.TargetTemperature <= 0 {
		return fmt.Errorf("target temperature must be positive")
	}
	if p.TargetPressure < 0 {
		return fmt.Errorf("target pressure must not be negative")
	}
	if p.FeedTarget < 0 {
		return fmt.Errorf("feed target must not be negative")
	}
	if p.CoolTarget < 0 {
		return fmt.Errorf("cool target must not be negative")
	}
	return nil
}

func ValidateCurve(c model.Curve) error {
	if c.ID == "" {
		return fmt.Errorf("curve id must not be empty")
	}
	seen := make(map[int]bool)
	for _, step := range c.Steps {
		if seen[step.Index] {
			return fmt.Errorf("duplicate curve step index %d", step.Index)
		}
		seen[step.Index] = true
		if step.Temperature <= 0 {
			return fmt.Errorf("curve step %d temperature must be positive", step.Index)
		}
		if step.Pressure < 0 {
			return fmt.Errorf("curve step %d pressure must not be negative", step.Index)
		}
	}
	return nil
}
