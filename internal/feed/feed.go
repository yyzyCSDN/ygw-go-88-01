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
	seq        []param.SequenceEntry
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


func (s *Service) State(id string) model.FeedState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state[id]
}

func (s *Service) SyncCurve() {
	s.mu.Lock()
	defer s.mu.Unlock()
	gen := s.store.Generation()
	if gen == s.curveGen {
		return
	}
	s.curveGen = gen
	params, ok := s.store.Params("main")
	if !ok {
		s.seq = nil
		return
	}
	curve, ok := s.store.Curve(params.CurveID)
	if !ok {
		s.seq = nil
		return
	}
	s.seq = param.BuildSequence(curve)
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

func (s *Service) StepAt(index int) (param.SequenceEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return param.CurrentStep(s.seq, index)
}
