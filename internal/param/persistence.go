package param

import (
	"encoding/json"
	"os"
	"path/filepath"

	"chemicalprocessdcs/internal/model"
)

type PersistedStore struct {
	Params map[string]model.ProcessParams `json:"params"`
	Curves map[string]model.Curve         `json:"curves"`
}

func (s *Store) Save(path string) error {
	payload := PersistedStore{
		Params: make(map[string]model.ProcessParams),
		Curves: make(map[string]model.Curve),
	}
	for id, p := range s.params {
		payload.Params[id] = p
	}
	for id, c := range s.curves {
		payload.Curves[id] = c
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (s *Store) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var payload PersistedStore
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	for id, p := range payload.Params {
		s.params[id] = p
	}
	for id, c := range payload.Curves {
		s.curves[id] = c
	}
	s.gen++
	return nil
}

func (s *Store) AllCurves() map[string]model.Curve {
	out := make(map[string]model.Curve, len(s.curves))
	for id, c := range s.curves {
		out[id] = c
	}
	return out
}
