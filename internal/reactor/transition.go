package reactor

import (
	"time"

	"chemicalprocessdcs/internal/model"
)

type Transition struct {
	ReactorID string
	From      model.ReactorState
	To        model.ReactorState
	At        time.Time
}

func (s *Service) recordTransitionLocked(id string, from, to model.ReactorState) {
	s.transitions = append(s.transitions, Transition{ReactorID: id, From: from, To: to, At: s.now().UTC()})
}

func (s *Service) TransitionCount(id string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, transition := range s.transitions {
		if transition.ReactorID == id {
			count++
		}
	}
	return count
}

func (s *Service) LastTransition(id string) (Transition, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.transitions) - 1; i >= 0; i-- {
		if s.transitions[i].ReactorID == id {
			return s.transitions[i], true
		}
	}
	return Transition{}, false
}
