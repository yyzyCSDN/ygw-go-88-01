package param

import (
	"sync"

	"chemicalprocessdcs/internal/model"
)

type Store struct {
	mu      sync.RWMutex
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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.params[id] = p
	s.gen++
}

func (s *Store) Params(id string) (model.ProcessParams, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.params[id]
	return p, ok
}

// paramsLocked returns the params for id. Caller must hold s.mu.
func (s *Store) paramsLocked(id string) (model.ProcessParams, bool) {
	p, ok := s.params[id]
	return p, ok
}

// AllParams returns a snapshot copy of all process params. The returned map is
// safe to use after the lock is released.
func (s *Store) AllParams() map[string]model.ProcessParams {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]model.ProcessParams, len(s.params))
	for id, p := range s.params {
		out[id] = p
	}
	return out
}

func (s *Store) Generation() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.gen
}

func (s *Store) SetCurve(c model.Curve) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.curves[c.ID] = c
	s.gen++
}

func (s *Store) Curve(id string) (model.Curve, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.curveLocked(id)
}

// curveLocked returns the curve for id. Caller must hold s.mu.
func (s *Store) curveLocked(id string) (model.Curve, bool) {
	c, ok := s.curves[id]
	return c, ok
}

// CurveSnapshot atomically captures the store generation together with the
// params for id and the curve those params reference, all under a single lock
// acquisition. Because the generation, the params, and the curve are read
// together, a concurrent SetParams/SetCurve (a "curve switch") can never
// interleave the reads: the returned pair always belongs to one coherent
// version of the store, never a mix of old params with a new curve or vice
// versa. Consumers that build a process sequence from (Params, Curve) are
// therefore guaranteed to see one complete version.
//
// If gen equals prevGen the snapshot is marked Unchanged and Params/Curve are
// not populated, so callers can skip a rebuild without re-reading the store.
type CurveSnapshot struct {
	Gen       uint64
	Params    model.ProcessParams
	Curve     model.Curve
	OK        bool
	Unchanged bool
}

func (s *Store) CurveSnapshot(id string, prevGen uint64) CurveSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap := CurveSnapshot{Gen: s.gen}
	if s.gen == prevGen {
		snap.Unchanged = true
		return snap
	}
	p, ok := s.params[id]
	if !ok {
		return snap
	}
	snap.Params = p
	c, ok := s.curves[p.CurveID]
	if !ok {
		return snap
	}
	snap.Curve = c
	snap.OK = true
	return snap
}
