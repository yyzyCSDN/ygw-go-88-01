package param

import "chemicalprocessdcs/internal/model"

type Recipe struct {
	ID       string
	CurveIDs []string
}

func (s *Store) SetRecipe(r Recipe) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.recipes == nil {
		s.recipes = make(map[string]Recipe)
	}
	s.recipes[r.ID] = r
	s.gen++
}

func (s *Store) Recipe(id string) (Recipe, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.recipes[id]
	return r, ok
}

func (s *Store) RecipeCurves(id string) ([]model.Curve, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	recipe, ok := s.recipes[id]
	if !ok {
		return nil, false
	}
	curves := make([]model.Curve, 0, len(recipe.CurveIDs))
	for _, curveID := range recipe.CurveIDs {
		if c, ok := s.curveLocked(curveID); ok {
			curves = append(curves, c)
		}
	}
	return curves, true
}

func (s *Store) AllRecipes() map[string]Recipe {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]Recipe, len(s.recipes))
	for id, r := range s.recipes {
		out[id] = r
	}
	return out
}
