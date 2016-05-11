package types

import "github.com/docker/engine-api/types/events"

type Message struct {
	Event  *events.Message
	UUID   string
	Action string
	Host   string
}
