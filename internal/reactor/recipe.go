package reactor

import "chemicalprocessdcs/internal/model"

type Recipe struct {
	CurveID   string
	StepIndex int
	Complete  bool
}

func (s *Service) Recipe(id string) (Recipe, error) {
	r, err := s.Get(id)
	if err != nil {
		return Recipe{}, err
	}
	params, ok := s.mainParams()
	if !ok {
		return Recipe{}, model.ErrParamNotFound
	}
	curve, ok := s.params.Curve(params.CurveID)
	if !ok {
		return Recipe{}, model.ErrCurveNotFound
	}
	index := s.feed.CurrentIndex()
	complete := r.State == model.ReactorIdle && index >= len(curve.Steps)
	return Recipe{CurveID: curve.ID, StepIndex: index, Complete: complete}, nil
}

