package verifier

import (
	"errors"
	"fmt"

	"github.com/rancher/go-rancher/client"
)

func (rvr *RancherK8sVerifiedResponse) PrepareResponse(verified bool, container *client.Container, c *client.RancherClient) error {

	rvr.verified = verified

	if _, ok := container.Labels["io.kubernetes.pod.namespace"].(string); !ok {
		return errors.New("No pod namespace found")
	}

	rvr.namespace = container.Labels["io.kubernetes.pod.namespace"].(string)

	project, err := getProjectFromAPIKey(c)
	if err != nil {
		return err
	}

	rvr.environmentName = project.Name

	rvr.labelPath = container.Labels["secrets.bridge.k8s.path"].(string)

	// This shouldn't happen if the New Verifier Factory was used.
	if rvr.id == "" {
		rvr.id = container.ExternalId
	}

	return nil
}

func (rvr *RancherK8sVerifiedResponse) Path() string {
	return fmt.Sprintf("%s/%s/%s/%s", rvr.environmentName, rvr.namespace, rvr.labelPath, rvr.id)
}

func (rvr *RancherK8sVerifiedResponse) Verified() bool {
	return rvr.verified
}

func (rvr *RancherK8sVerifiedResponse) ID() string {
	return rvr.id
}
