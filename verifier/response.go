package verifier

import "fmt"

type VerifiedResponse interface {
	Path() string
	Verified() bool
	ID() string
}

type RancherVerifiedResponse struct {
	verified        bool
	serviceName     string
	stackName       string
	containerName   string
	environmentName string
	id              string
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
