package feed

import (
	"context"
	"sync"
	"time"

	"chemicalprocessdcs/internal/model"
	"chemicalprocessdcs/internal/param"
)

type Service struct {
	mu         sync.Mutex
	state      map[string]model.FeedState
	amount     map[string]float64
	timeout    time.Duration
	store      *param.Store
	curveGen   uint64
	// seq is the currently active process sequence. It is only mutated while
	// holding mu, so every reader that observes it under mu sees one coherent,
	// fully built version — never a half-published mix of two sequences.
	seq        []param.SequenceEntry
	curveID    string
	index      int
	cascadeMap map[string]Cascade
	history    []ConfirmationRecord
	snapshot   map[string]model.FeedState
	confirm    func(ctx context.Context, id string) (model.FeedConfirmation, error)
	now        func() time.Time
}

func NewService(store *param.Store, confirm func(ctx context.Context, id string) (model.FeedConfirmation, error)) *Service {
	return &Service{
		state:    make(map[string]model.FeedState),
		amount:   make(map[string]float64),
		snapshot: make(map[string]model.FeedState),
		timeout:  time.Duration(model.DefaultControlConfig().FeedTimeout) * time.Second,
		store:    store,
		confirm:  confirm,
		now:      time.Now,
	}
}

func (s *Service) Register(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.state[id]; !ok {
		s.state[id] = model.FeedIdle
		s.amount[id] = 0
	}
	s.snapshot[id] = model.FeedIdle
}

func (s *Service) RegisteredCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.snapshot)
}

func (s *Service) Start(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state[id] == model.FeedMetering || s.state[id] == model.FeedFeeding {
		return model.ErrFeedBusy
	}
	s.state[id] = model.FeedMetering
	return nil
}

func (s *Service) Meter(id string) error {
	s.mu.Lock()
	if s.state[id] != model.FeedMetering {
		s.mu.Unlock()
		return model.ErrFeedNotMetering
	}
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()
	conf, err := s.confirm(ctx, id)
	if err != nil {
		s.resetMeter(id)
		return err
	}
	if !conf.OK {
		s.resetMeter(id)
		return model.ErrFeedBusy
	}

	s.mu.Lock()
	s.state[id] = model.FeedFeeding
	s.amount[id] = conf.Amount
	s.recordConfirmation(id, conf)
	s.mu.Unlock()
	return nil
}

func (s *Service) resetMeter(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state[id] = model.FeedIdle
	s.index = 0
	s.amount[id] = 0
}

func (s *Service) Stop(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state[id] = model.FeedStopped
}

func (s *Service) Reset(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state[id] != model.FeedFeeding {
		s.state[id] = model.FeedIdle
	}
}

func (s *Service) Cancel(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state[id] = model.FeedIdle
	s.index = 0
	s.amount[id] = 0
}

func (s *Service) State(id string) model.FeedState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state[id]
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
	// sequence is one complete, coherent version. The cursor is reset
	// together with the sequence under the same lock, so index and seq are
	// never split across two versions either.
	snap := s.store.CurveSnapshot("main", s.curveGen)
	if snap.Unchanged {
		return
	}
	s.curveGen = snap.Gen
	if !snap.OK {
		s.seq = nil
		s.curveID = ""
		s.index = 0
		return
	}
	s.seq = param.BuildSequence(snap.Curve)
	s.curveID = snap.Curve.ID
	s.index = 0
}

func (s *Service) FeedTarget() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	params, ok := s.store.Params("main")
	if !ok {
		return 0
	}
	return params.FeedTarget
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

// Progress returns the active curve id, the feed cursor and the active
// sequence length together, taken under a single acquisition of s.mu and
// from one coherent version of the sequence (after syncCurveLocked). Callers
// that derive state from (curveID, index, length) — e.g. a "complete" check of
// index >= length — are therefore guaranteed the three values belong to the
// same version and never straddle a curve switch.
func (s *Service) Progress() (curveID string, index int, length int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncCurveLocked()
	return s.curveID, s.index, len(s.seq)
}
