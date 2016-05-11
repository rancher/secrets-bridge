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
	})
	if err != nil {
		return nil, err
	}

	return &RancherVerifier{
		client: client,
	}, nil
}

func (c *RancherVerifier) Verify(msg *types.Message) (VerifiedResponse, error) {
	resp := &RancherVerifiedResponse{}
	logrus.Infof("Verifing: %s", msg.UUID)
	logrus.Debugf("Verifing: %s", msg.Action)
	logrus.Debugf("Verifing: %s", msg.Host)

	container, err := c.requestCompleteContainerFromRancher(msg.UUID)
	if err != nil {
		return resp, err
	}

	if matchInfo(msg, container) {
		resp, err = c.prepareResponse(true, &container)
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

func matchInfo(msg *types.Message, container client.Container) bool {
	isVerified := false

	logrus.Debugf("rancher ext id: %s for eventId: %s", container.ExternalId, msg.Event.ID)

	if msg.Event.ID == container.ExternalId {
		isVerified = true
	}

	logrus.Debugf("Is Verified? %v", isVerified)

	return isVerified
}

func (c *RancherVerifier) requestCompleteContainerFromRancher(uuid string) (client.Container, error) {
	maxWaitTime := 60 * time.Second
	var container client.Container

	for i := 1 * time.Second; i < maxWaitTime; i *= time.Duration(2) {
		containers, err := c.client.Container.List(&client.ListOpts{
			Filters: map[string]interface{}{
				"uuid": uuid,
			},
		})
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

func (c *RancherVerifier) prepareResponse(verified bool, container *client.Container) (*RancherVerifiedResponse, error) {
	svc, err := getServiceFromContainer(c.client, container)
	if err != nil {
		return nil, err
	}

	stk, err := getStackFromService(c.client, svc)
	if err != nil {
		return nil, err
	}

	env, err := getEnvFromStack(c.client, stk)
	if err != nil {
		return nil, err
	}

	// Should probably get a New or Init method...
	resp := &RancherVerifiedResponse{
		verified:        verified,
		serviceName:     svc.Name,
		stackName:       stk.Name,
		environmentName: env.Name,
		containerName:   container.Name,
		id:              container.ExternalId,
	}

	return resp, nil
}

func getServiceFromContainer(c *client.RancherClient, container *client.Container) (*client.Service, error) {
	var svc *client.ServiceCollection
	err := c.GetLink(container.Resource, "services", &svc)
	if err != nil || len(svc.Data) == 0 {
		return nil, err
	}

	return &svc.Data[0], nil
}

func getStackFromService(c *client.RancherClient, service *client.Service) (*client.Environment, error) {
	var stack *client.Environment
	err := c.GetLink(service.Resource, "environment", &stack)
	if err != nil || stack.Name == "" {
		return nil, err
	}
	return stack, nil
}

func getEnvFromStack(c *client.RancherClient, stk *client.Environment) (*client.Project, error) {
	var environment *client.Project
	err := c.GetLink(stk.Resource, "account", &environment)
	if err != nil || environment.Name == "" {
		return nil, err
	}
	return environment, nil
}
