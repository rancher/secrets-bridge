package agent

import (
	"strings"
	"time"

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
	nameKey := "name"
	verifyKey := "io.rancher.container.uuid"

	if cem.ContainerType == "kubernetes" {
		verifyKey = "io.kubernetes.pod.namespace"
		nameKey = "io.kubernetes.pod.name"
	}

	name := cem.Event.Actor.Attributes[nameKey]

	logrus.Debugf("Received: %s as a container name", name)

	if strings.HasPrefix(name, "r-") {
		name = strings.Replace(name, "r-", "", 1)
		nameSplit := strings.Split(name, "-")

		name = strings.Join(nameSplit[:len(nameSplit)-1], "-")
	}

	logrus.Debugf("Using: %s as a container name", name)

	container := metadata.Container{}
	for len(container.Labels) == 0 {
		containers, err := mdCli.GetContainers()
		if err != nil {
			return err
		}
		container = loopContainers(name, verifyKey, cem.Event.Actor.Attributes[verifyKey], containers)
		time.Sleep(100 * time.Millisecond)
	}

	logrus.Debugf("UUID: %s found", container.UUID)
	cem.UUID = container.UUID

	return nil
}

func loopContainers(name, vKey, expectedVKeyValue string, containers []metadata.Container) metadata.Container {
	for _, container := range containers {
		if container.Name == name {
			if value, ok := container.Labels[vKey]; ok {
				if value == expectedVKeyValue {
					return container
				}
			}
		}
	}

	return metadata.Container{}

}
