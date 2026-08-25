package record

import "chemicalprocessdcs/internal/model"

func StatisticsOf(entries []model.RecordEntry) (average float64, peakTemperature float64, peakPressure float64) {
	if len(entries) == 0 {
		return 0, 0, 0
	}
	var sum float64
	for _, entry := range entries {
		sum += entry.Temperature
		if entry.Temperature > peakTemperature {
			peakTemperature = entry.Temperature
		}
		if entry.Pressure > peakPressure {
			peakPressure = entry.Pressure
		}
	}
	return sum / float64(len(entries)), peakTemperature, peakPressure
}
