package cool

func CoolingRampTarget(current float64, target float64, step int) float64 {
	if step <= 0 {
		return current
	}
	delta := (current - target) / float64(step)
	if delta < 0 {
		return target
	}
	return current - delta
}

func (s *Service) RampTo(target float64, steps int) float64 {
	// Snapshot the sequence atomically so current is read from one coherent
	// version even while SyncCurve may be replacing the active sequence.
	seq, _ := s.snapshotSeq()
	current := 0.0
	if len(seq) > 0 {
		current = seq[len(seq)-1].Temperature
	}
	return CoolingRampTarget(current, target, steps)
}
