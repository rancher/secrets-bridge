package verifier

import (
	"errors"

	"github.com/rancher/go-rancher/client"
)

func getProjectFromAPIKey(c *client.RancherClient) (*client.Project, error) {
	projects, err := c.Project.List(&client.ListOpts{})
	if err != nil {
		return nil, err
	}

	if len(projects.Data) == 0 {
		return nil, errors.New("No project found for key")
	}

	return &projects.Data[0], nil
}

func getServiceFromContainer(c *client.RancherClient, container *client.Container) (*client.Service, error) {
	var svc *client.ServiceCollection
	err := c.GetLink(container.Resource, "services", &svc)
	if err != nil {
		return nil, err
	}
	if len(svc.Data) == 0 {
		return nil, errors.New("Error: This container is not running inside a Rancher service")
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
