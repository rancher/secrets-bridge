package types

import "github.com/docker/engine-api/types/events"

type Message struct {
	Event         *events.Message
	UUID          string `json:"UUID"`
	Action        string `json:"Action"`
	Host          string `json:"Host"`
	ContainerType string `json:"container_type"`
}
