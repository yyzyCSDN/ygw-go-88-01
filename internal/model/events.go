package model

import "time"

type Event struct {
	ReactorID string
	Kind      string
	Message   string
	At        time.Time
}

func NewEvent(reactorID string, kind string, message string, at time.Time) Event {
	return Event{ReactorID: reactorID, Kind: kind, Message: message, At: at}
}
