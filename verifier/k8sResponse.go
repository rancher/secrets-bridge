package verifier

import "fmt"

type RancherK8sVerifiedResponse struct {
	verified        bool
	namespace       string
	environmentName string
	labelPath       string
	id              string
}

func (rvr *RancherK8sVerifiedResponse) Path() string {
	return fmt.Sprintf("%s/%s/%s", rvr.environmentName, rvr.namespace, rvr.labelPath)
}

func (rvr *RancherK8sVerifiedResponse) Verified() bool {
	return rvr.verified
}

func (rvr *RancherK8sVerifiedResponse) ID() string {
	return rvr.id
}
