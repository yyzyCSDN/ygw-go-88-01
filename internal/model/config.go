package model

import "fmt"

type ControlConfig struct {
	StableTolerance float64
	StableTimeout   int
	FeedTimeout     int
	MaxPressure     float64
	MaxTemperature  float64
}

func DefaultControlConfig() ControlConfig {
	return ControlConfig{
		StableTolerance: 1.5,
		StableTimeout:   12,
		FeedTimeout:     8,
		MaxPressure:     12.0,
		MaxTemperature:  280.0,
	}
}

func ValidateControlConfig(cfg ControlConfig) error {
	if cfg.StableTolerance <= 0 {
		return fmt.Errorf("stable tolerance must be positive")
	}
	if cfg.StableTimeout <= 0 {
		return fmt.Errorf("stable timeout must be positive")
	}
	if cfg.FeedTimeout <= 0 {
		return fmt.Errorf("feed timeout must be positive")
	}
	if cfg.MaxPressure <= 0 {
		return fmt.Errorf("max pressure must be positive")
	}
	if cfg.MaxTemperature <= 0 {
		return fmt.Errorf("max temperature must be positive")
	}
	if cfg.MaxTemperature <= cfg.StableTolerance {
		return fmt.Errorf("max temperature must exceed stable tolerance")
	}
	return nil
}
