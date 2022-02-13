package structures

import "time"

type EventsSubscribeUser struct {
	User      User      `json:"user"`
	Timestamp time.Time `json:"timestamp"`
}

type EventsSubscribeChat struct {
	User      User      `json:"user"`
	Message   Message   `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}
