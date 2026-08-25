package alarm

import (
	"sort"
	"time"
)

type HistoryEntry struct {
	ReactorID string
	Level     string
	Message   string
	At        time.Time
}

func (s *Service) appendHistoryLocked(entry HistoryEntry) {
	if s.history == nil {
		s.history = make([]HistoryEntry, 0)
	}
	s.history = append(s.history, entry)
}

func (s *Service) History() []HistoryEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]HistoryEntry, len(s.history))
	copy(out, s.history)
	sort.Slice(out, func(i, j int) bool { return out[i].At.Before(out[j].At) })
	return out
}

func (s *Service) ActiveCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, a := range s.alarms {
		if a.Active {
			count++
		}
	}
	return count
}

func (s *Service) Acknowledge(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.alarms[id]
	if !ok {
		return
	}
	a.Active = false
	s.alarms[id] = a
	s.appendHistoryLocked(HistoryEntry{ReactorID: id, Level: a.Level, Message: "acknowledged", At: time.Now().UTC()})
}
