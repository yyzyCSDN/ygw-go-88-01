package alarm

import (
	"time"

	"chemicalprocessdcs/internal/model"
)

func (s *Service) Restore(id string) error {
	if err := s.esd.MarkRestored(id); err != nil {
		return err
	}
	s.mu.Lock()
	s.alarms[id] = model.Alarm{ReactorID: id, Level: "info", Message: "restored", Active: false}
	s.appendHistoryLocked(HistoryEntry{ReactorID: id, Level: "info", Message: "restored", At: time.Now().UTC()})
	s.mu.Unlock()
	return nil
}

func (s *Service) RecoverSnapshot() map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]bool, len(s.snapshot))
	for id, restored := range s.snapshot {
		out[id] = restored
	}
	return out
}
