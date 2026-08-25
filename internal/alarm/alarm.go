package alarm

import (
	"sync"
	"time"

	"chemicalprocessdcs/internal/esd"
	"chemicalprocessdcs/internal/model"
)

type Service struct {
	mu       sync.Mutex
	alarms   map[string]model.Alarm
	esd      *esd.Service
	snapshot map[string]bool
	history  []HistoryEntry
}

func NewService(esdSvc *esd.Service) *Service {
	return &Service{
		alarms:   make(map[string]model.Alarm),
		esd:      esdSvc,
		snapshot: make(map[string]bool),
	}
}

func (s *Service) ReportTrip(id string, err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 停车失败时给出明确的失败告警，避免与"已停车"混淆。
	if err != nil {
		msg := "reactor trip failed: " + err.Error()
		s.alarms[id] = model.Alarm{ReactorID: id, Level: "critical", Message: msg, Active: true}
		s.appendHistoryLocked(HistoryEntry{ReactorID: id, Level: "critical", Message: msg, At: time.Now().UTC()})
		return nil
	}
	s.alarms[id] = model.Alarm{ReactorID: id, Level: "critical", Message: "reactor tripped", Active: true}
	s.appendHistoryLocked(HistoryEntry{ReactorID: id, Level: "critical", Message: "reactor tripped", At: time.Now().UTC()})
	return nil
}

