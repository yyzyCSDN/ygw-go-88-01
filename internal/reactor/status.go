package reactor

import "chemicalprocessdcs/internal/model"

type Status struct {
	ID          string             `json:"id"`
	State       model.ReactorState `json:"state"`
	Temperature float64            `json:"temperature"`
	Pressure    float64            `json:"pressure"`
	Tripped     bool               `json:"tripped"`
	Restored    bool               `json:"restored"`
}

func (s *Service) Status(id string) (Status, error) {
	r, err := s.Get(id)
	if err != nil {
		return Status{}, err
	}
	return Status{
		ID:          r.ID,
		State:       r.State,
		Temperature: r.Temperature,
		Pressure:    r.Pressure,
		Tripped:     r.Tripped,
		Restored:    r.Restored,
	}, nil
}

func (s *Service) AllStatuses() []Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Status, 0, len(s.reactors))
	for _, r := range s.reactors {
		out = append(out, Status{
			ID:          r.ID,
			State:       r.State,
			Temperature: r.Temperature,
			Pressure:    r.Pressure,
			Tripped:     r.Tripped,
			Restored:    r.Restored,
		})
	}
	return out
}
