package agent

import (
	"errors"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types/events"
	"github.com/rancher/go-rancher-metadata/metadata"
)

type ContainerEventMessage struct {
	Event         *events.Message
	UUID          string `json:"UUID"`
	Action        string `json:"Action"`
	Host          string `json:"Host"`
	ContainerType string `json:"container_type"`
}

func (cem *ContainerEventMessage) SetUUIDFromMetadata(mdCli *metadata.Client) error {
	var uuid string
	nameKey := "name"
	verifyKey := "io.rancher.container.uuid"

	if cem.ContainerType == "kubernetes" {
		verifyKey = "io.kubernetes.pod.namespace"
		nameKey = "io.kubernetes.pod.name"
	}

	name := cem.Event.Actor.Attributes[nameKey]

	logrus.Debugf("Received: %s as a container name", name)

	name = strings.Replace(name, "r-", "", 1)
	logrus.Debugf("Using: %s as a container name", name)

	containers, err := mdCli.GetContainers()
	if err != nil {
		logrus.Errorf("Failed to get containers: %s", err)
		return err
	}

	for _, container := range containers {
		if container.Name == name {
			if value, ok := container.Labels[verifyKey]; ok {
				if value == cem.Event.Actor.Attributes[verifyKey] {
					uuid = container.UUID
					break
				}
			}
		}
	}

	if uuid == "" {
		logrus.Debugf("No UUID Found")
		return errors.New("No UUID found")
	}

	logrus.Debugf("UUID: %s found", uuid)
	cem.UUID = uuid

	return nil
}
