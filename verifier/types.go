package verifier

import (
	"errors"

	"github.com/rancher/go-rancher/client"
	"github.com/rancher/secrets-bridge/types"
)

type VerifiedResponse interface {
	Path() string
	Verified() bool
	ID() string
	PrepareResponse(bool, *client.Container, *client.RancherClient) error
}

func NewVerifiedResponse(msg *types.Message) (VerifiedResponse, error) {
	switch msg.ContainerType {
	case "kubernetes":
		return &RancherK8sVerifiedResponse{id: msg.Event.ID}, nil
	case "cattle":
		return &RancherVerifiedResponse{}, nil
	default:
		return nil, errors.New("Invalid Type")
	}
}

type RancherVerifiedResponse struct {
	verified        bool
	serviceName     string
	stackName       string
	containerName   string
	environmentName string
	id              string
}

type RancherK8sVerifiedResponse struct {
	verified        bool
	namespace       string
	environmentName string
	labelPath       string
	id              string
}
