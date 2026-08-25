package param

import "chemicalprocessdcs/internal/model"

type SequenceEntry struct {
	Step        int
	Temperature float64
	Pressure    float64
}

func BuildSequence(curve model.Curve) []SequenceEntry {
	entries := make([]SequenceEntry, 0, len(curve.Steps))
	for _, step := range curve.Steps {
		entries = append(entries, SequenceEntry{
			Step:        step.Index,
			Temperature: step.Temperature,
			Pressure:    step.Pressure,
		})
	}
	return entries
}

func CurrentStep(entries []SequenceEntry, index int) (SequenceEntry, bool) {
	if index < 0 || index >= len(entries) {
		return SequenceEntry{}, false
	}
	return entries[index], true
}
