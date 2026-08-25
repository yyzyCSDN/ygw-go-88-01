package feed

import "chemicalprocessdcs/internal/model"

type ConfirmationRecord struct {
	ReactorID string
	Amount    float64
	OK        bool
}

func (s *Service) recordConfirmation(id string, conf model.FeedConfirmation) {
	if s.history == nil {
		s.history = make([]ConfirmationRecord, 0)
	}
	s.history = append(s.history, ConfirmationRecord{ReactorID: id, Amount: conf.Amount, OK: conf.OK})
}

func (s *Service) ConfirmationCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.history)
}
