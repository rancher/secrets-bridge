package verifier

import (
	"errors"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/client"
	"github.com/rancher/secrets-bridge/types"
)

type VerifierConfig struct {
	RancherUrl       string
	rancherAccessKey string
	rancherSecretKey string
}

type Verifier interface {
	Verify(*types.Message) (VerifiedResponse, error)
}

type AuthVerifier interface {
	VerifyAuth(string) (bool, error)
}

type RancherVerifier struct {
	client *client.RancherClient
}

func NewConfig(url, access, secret string) *VerifierConfig {
	return &VerifierConfig{
		RancherUrl:       url,
		rancherAccessKey: access,
		rancherSecretKey: secret,
	}
}

func NewVerifier(name string, config *VerifierConfig) (Verifier, error) {
	return NewRancherVerifier(config)
}

func NewAuthVerifier(name string, config *VerifierConfig) (AuthVerifier, error) {
	return NewRancherVerifier(config)
}

func NewRancherVerifier(config *VerifierConfig) (*RancherVerifier, error) {
	client, err := client.NewRancherClient(&client.ClientOpts{
		Url:       config.RancherUrl,
		AccessKey: config.rancherAccessKey,
		SecretKey: config.rancherSecretKey,
		Timeout:   10 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return &RancherVerifier{
		client: client,
	}, nil
}

func (c *RancherVerifier) Verify(msg *types.Message) (VerifiedResponse, error) {
	resp, _ := NewVerifiedResponse(msg)

	logrus.Infof("Verifing: %s", msg.UUID)
	logrus.Debugf("Verifing: %s", msg.Action)
	logrus.Debugf("Verifing: %s", msg.Host)
	logrus.Debugf("Verifing: %s", msg.ContainerType)

	container, err := c.requestCompleteContainerFromRancher(msg.UUID)
	if err != nil {
		return resp, err
	}

	if c.matchInfo(msg, container) {
		err = resp.PrepareResponse(true, &container, c.client)
		if err != nil {
			return resp, err
		}
	}

	return resp, nil
}

func (c *RancherVerifier) VerifyAuth(authString string) (bool, error) {
	verified := false
	if len(authString) <= 0 {
		return verified, errors.New("No token found")
	}
	split := strings.SplitN(authString, ":", 3)

	if len(split) != 3 {
		return verified, errors.New("Malformed token")
	}

	verified = true

	logrus.Debugf("UUID: %s", split[0])
	logrus.Debugf("Timestamp: %s", split[1])
	logrus.Debugf("HMAC: %x", split[2])

	return verified, nil
}

func (c *RancherVerifier) matchInfo(msg *types.Message, container client.Container) bool {
	switch msg.ContainerType {
	case "cattle":
		return matchInfoCattle(msg, container)
	case "kubernetes":
		return c.matchInfoK8s(msg, container)
	}
	return false
}

func (c *RancherVerifier) matchInfoK8s(msg *types.Message, container client.Container) bool {
	isVerified := false

	logrus.Debugf("rancher k8s pod uid: %s for eventId: %s", container.Labels["io.kubernetes.pod.uid"], msg.Event.ID)

	eventIdContainer, err := c.requestContainer(&client.ListOpts{
		Filters: map[string]interface{}{
			"externalId": msg.Event.ID,
		},
	})
	if err != nil {
		return false
	}

	if container.Labels["io.kubernetes.pod.uid"] == eventIdContainer.Labels["io.kubernetes.pod.uid"] {
		logrus.Debugf("Pod UUID: %s and %s match", container.Labels["io.kubernetes.pod.uid"], eventIdContainer.Labels["io.kubernetes.pod.uid"])
		isVerified = true
	}

	return isVerified
}

func matchInfoCattle(msg *types.Message, container client.Container) bool {
	isVerified := false

	logrus.Debugf("rancher ext id: %s for eventId: %s", container.ExternalId, msg.Event.ID)

	if msg.Event.ID == container.ExternalId {
		isVerified = true
	}

	logrus.Debugf("Is Verified? %v", isVerified)

	return isVerified
}

func (c *RancherVerifier) requestCompleteContainerFromRancher(uuid string) (client.Container, error) {
	listOpts := &client.ListOpts{
		Filters: map[string]interface{}{
			"uuid": uuid,
		},
	}
	return c.requestContainer(listOpts)
}

func (c *RancherVerifier) requestContainer(opts *client.ListOpts) (client.Container, error) {
	maxWaitTime := 60 * time.Second
	var container client.Container

	for i := 1 * time.Second; i < maxWaitTime; i *= time.Duration(2) {
		containers, err := c.client.Container.List(opts)
		if err != nil {
			return client.Container{}, err
		}

		if len(containers.Data) > 0 {
			container = containers.Data[0]
		}

		if container.ExternalId == "" {
			time.Sleep(i)
		} else {
			return container, nil
		}
	}

	return container, nil
}
