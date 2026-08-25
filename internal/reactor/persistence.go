package reactor

import (
	"encoding/json"
	"os"
	"path/filepath"

	"chemicalprocessdcs/internal/model"
)

type PersistedState struct {
	Reactors map[string]model.Reactor `json:"reactors"`
}

func (s *Service) SaveState(path string) error {
	snapshot := s.TakeSnapshot()
	data, err := json.MarshalIndent(PersistedState{Reactors: snapshot.Reactors}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (s *Service) LoadState(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var persisted PersistedState
	if err := json.Unmarshal(data, &persisted); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, r := range persisted.Reactors {
		cp := r
		s.reactors[id] = &cp
		s.feed.Register(id)
	}
	return nil
}
