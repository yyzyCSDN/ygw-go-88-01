package reactor

import (
	"sync"
	"time"

	"chemicalprocessdcs/internal/alarm"
	"chemicalprocessdcs/internal/cool"
	"chemicalprocessdcs/internal/esd"
	"chemicalprocessdcs/internal/feed"
	"chemicalprocessdcs/internal/model"
	"chemicalprocessdcs/internal/param"
	"chemicalprocessdcs/internal/record"
)

type Service struct {
	mu             sync.Mutex
	reactors       map[string]*model.Reactor
	params         *param.Store
	cool           *cool.Service
	feed           *feed.Service
	esd            *esd.Service
	alarm          *alarm.Service
	rec            *record.Service
	cfg            model.ControlConfig
	paramGen       uint64
	paramCache     map[string]model.ProcessParams
	trends         map[string]*model.Trend
	cachedRestored map[string]bool
	transitions    []Transition
	now            func() time.Time
}

func NewService(
	params *param.Store,
	coolSvc *cool.Service,
	feedSvc *feed.Service,
	esdSvc *esd.Service,
	alarmSvc *alarm.Service,
	recSvc *record.Service,
) *Service {
	return &Service{
		reactors:       make(map[string]*model.Reactor),
		params:         params,
		cool:           coolSvc,
		feed:           feedSvc,
		esd:            esdSvc,
		alarm:          alarmSvc,
		rec:            recSvc,
		cfg:            model.DefaultControlConfig(),
		paramCache:     make(map[string]model.ProcessParams),
		trends:         make(map[string]*model.Trend),
		cachedRestored: make(map[string]bool),
		now:            time.Now,
	}
}

func (s *Service) Register(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.reactors[id]; ok {
		return
	}
	s.reactors[id] = &model.Reactor{ID: id, State: model.ReactorIdle}
	s.feed.Register(id)
}

func (s *Service) Get(id string) (*model.Reactor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.reactors[id]
	if !ok {
		return nil, nil
	}
	return r, nil
}

func (s *Service) SetReading(id string, temperature, pressure float64) error {
	if err := ValidateReading(temperature, pressure); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.reactors[id]
	if !ok {
		return model.ErrReactorNotFound
	}
	r.Temperature = temperature
	r.Pressure = pressure
	s.recordTrendLocked(id, temperature, pressure)
	return nil
}

func (s *Service) recordTrendLocked(id string, temperature, pressure float64) {
	trend, ok := s.trends[id]
	if !ok {
		trend = &model.Trend{ReactorID: id}
		s.trends[id] = trend
	}
	trend.Add(model.TrendPoint{Temperature: temperature, Pressure: pressure})
}

func (s *Service) syncParamsLocked() {
	gen := s.params.Generation()
	if gen == s.paramGen {
		return
	}
	s.paramGen = gen
	next := make(map[string]model.ProcessParams)
	if p, ok := s.params.Params("main"); ok {
		next["main"] = p
	}
	s.paramCache = next
}

func (s *Service) mainParams() (model.ProcessParams, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncParamsLocked()
	p, ok := s.paramCache["main"]
	return p, ok
}

func (s *Service) Temperature(id string) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.reactors[id]
	if !ok {
		return 0, model.ErrReactorNotFound
	}
	return r.Temperature, nil
}

func (s *Service) ReactorCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.reactors)
}

func (s *Service) SetConfig(cfg model.ControlConfig) {
	if model.ValidateControlConfig(cfg) != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
}
