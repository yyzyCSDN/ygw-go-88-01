package cool

import (
	"sync"

	"chemicalprocessdcs/internal/model"
	"chemicalprocessdcs/internal/param"
)

type Service struct {
	mu       sync.Mutex
	active   map[string]bool
	curveGen uint64
	// seq is the currently active process sequence. It is only mutated while
	// holding mu, so every reader that observes it under mu sees one coherent,
	// fully built version — never a half-published mix of two sequences.
	seq     []param.SequenceEntry
	version uint64
	store   *param.Store
	valve   func(id string, open bool) error
}

func NewService(store *param.Store, valve func(id string, open bool) error) *Service {
	return &Service{
		active: make(map[string]bool),
		store:  store,
		valve:  valve,
	}
}

// SyncCurve rebuilds the local sequence from the store's current curve. The
// build (params/curve lookup + BuildSequence) and the publication of the new
// seq happen atomically under s.mu, so a consumer can never read a sequence
// that is half-old / half-new.
func (s *Service) SyncCurve() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncCurveLocked()
}

// syncCurveLocked is the lock-held implementation of SyncCurve.
func (s *Service) syncCurveLocked() {
	// Read the generation, the params, and the referenced curve from the store
	// as one atomic snapshot (a single store-level lock acquisition). This is
	// the fix for the curve-switch race: previously Generation, Params and
	// Curve were three separate lock acquisitions, so a concurrent curve
	// switch (SetParams/SetCurve bumping the generation) could interleave them
	// and BuildSequence would be fed old params stitched to a new curve (or
	// the reverse) — half-old / half-new steps. The snapshot guarantees the
	// params and curve always belong to the same store version, so the rebuilt
	// sequence is one complete, coherent version.
	snap := s.store.CurveSnapshot("main", s.curveGen)
	if snap.Unchanged {
		return
	}
	s.curveGen = snap.Gen
	if !snap.OK {
		s.seq = nil
		return
	}
	s.seq = param.BuildSequence(snap.Curve)
	s.version++
}

// snapshotSeq returns a copy of the current sequence taken atomically under
// s.mu. Callers iterate their private copy and are therefore guaranteed to
// read one complete version even if SyncCurve replaces the active sequence
// concurrently. Returns the current version too.
func (s *Service) snapshotSeq() ([]param.SequenceEntry, uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Always reconcile against the store generation so a consumer that did not
	// call SyncCurve first still observes the latest coherent version.
	s.syncCurveLocked()
	out := make([]param.SequenceEntry, len(s.seq))
	copy(out, s.seq)
	return out, s.version
}

func (s *Service) Version() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.version
}

func (s *Service) Cool(id string, stepIndex int) error {
	entry, ok := s.StepAt(stepIndex)
	if !ok {
		return model.ErrCurveNotFound
	}
	if err := s.valve(id, true); err != nil {
		return err
	}
	zones := s.ZonesFor(entry.Temperature, entry.Temperature*0.5, 2)
	_ = zones
	s.mu.Lock()
	s.active[id] = true
	s.mu.Unlock()
	_ = entry
	return nil
}

func (s *Service) Stop(id string) error {
	return s.stopWithRetry(id, 3)
}

func (s *Service) stopWithRetry(id string, attempts int) error {
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if err := s.valve(id, false); err != nil {
			lastErr = err
			continue
		}
		s.mu.Lock()
		s.active[id] = false
		s.mu.Unlock()
		return nil
	}
	return lastErr
}

func (s *Service) Active(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active[id]
}

// StepAt returns the entry at index, taken from a coherent snapshot of the
// active sequence. The read happens entirely under s.mu, so it never straddles
// a SyncCurve replacement.
func (s *Service) StepAt(index int) (param.SequenceEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncCurveLocked()
	return param.CurrentStep(s.seq, index)
}

func (s *Service) SequenceLength() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncCurveLocked()
	return len(s.seq)
}
