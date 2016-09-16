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

	if namespace, ok := container.Labels["io.kubernetes.pod.namespace"].(string); ok {
		rvr.namespace = namespace
	}

	project, err := getProjectFromAPIKey(c)
	if err != nil {
		return err
	}

	rvr.environmentName = project.Name

	if labelPath, ok := container.Labels["secrets.bridge.k8s.path"].(string); ok {
		rvr.labelPath = labelPath
	}

	// This shouldn't happen if the New Verifier Factory was used.
	if rvr.id == "" {
		rvr.id = container.ExternalId
	}

	return nil
}

func (rvr *RancherK8sVerifiedResponse) Path() string {
	if rvr.labelPath == "" {
		return fmt.Sprintf("%s/%s/%s", rvr.environmentName, rvr.namespace, rvr.id)
	}

	return fmt.Sprintf("%s/%s/%s/%s", rvr.environmentName, rvr.namespace, rvr.labelPath, rvr.id)
}

func (rvr *RancherK8sVerifiedResponse) Verified() bool {
	return rvr.verified
}

func (rvr *RancherK8sVerifiedResponse) ID() string {
	return rvr.id
}
