package param

import "chemicalprocessdcs/internal/model"

type Store struct {
	params  map[string]model.ProcessParams
	curves  map[string]model.Curve
	recipes map[string]Recipe
	gen     uint64
}

func NewStore() *Store {
	return &Store{
		params:  make(map[string]model.ProcessParams),
		curves:  make(map[string]model.Curve),
		recipes: make(map[string]Recipe),
	}
}

func (s *Store) SetParams(id string, p model.ProcessParams) {
	s.params[id] = p
	s.gen++
}

func (s *Store) Params(id string) (model.ProcessParams, bool) {
	p, ok := s.params[id]
	return p, ok
}

func (s *Store) Generation() uint64 {
	return s.gen
}

func (s *Store) SetCurve(c model.Curve) {
	s.curves[c.ID] = c
	s.gen++
}

func (s *Store) Curve(id string) (model.Curve, bool) {
	c, ok := s.curves[id]
	return c, ok
}
