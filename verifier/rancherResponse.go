package verifier

import (
	"fmt"

	"github.com/rancher/go-rancher/client"
)

func (rvr *RancherVerifiedResponse) PrepareResponse(verified bool, container *client.Container, c *client.RancherClient) error {
	svc, err := getServiceFromContainer(c, container)
	if err != nil {
		return err
	}

	stk, err := getStackFromService(c, svc)
	if err != nil {
		return err
	}

	env, err := getEnvFromStack(c, stk)
	if err != nil {
		return err
	}

	rvr.verified = verified
	rvr.serviceName = svc.Name
	rvr.stackName = stk.Name
	rvr.environmentName = env.Name
	rvr.containerName = container.Name
	rvr.id = container.ExternalId

	return nil
}

func (rvr *RancherVerifiedResponse) Path() string {
	return fmt.Sprintf("%s/%s/%s/%s", rvr.environmentName, rvr.stackName, rvr.serviceName, rvr.containerName)
}

func (rvr *RancherVerifiedResponse) Verified() bool {
	return rvr.verified
}

func (rvr *RancherVerifiedResponse) ID() string {
	return rvr.id
}
