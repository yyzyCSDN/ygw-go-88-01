package alarm

import (
	"time"

	"chemicalprocessdcs/internal/model"
)

func (s *Service) Restore(id string) error {
	_ = s.esd.MarkRestored(id)
	s.mu.Lock()
	s.alarms[id] = model.Alarm{ReactorID: id, Level: "info", Message: "restored", Active: false}
	s.snapshot[id] = true
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
