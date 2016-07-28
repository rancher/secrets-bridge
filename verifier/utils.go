package verifier

import (
	"github.com/rancher/go-rancher/client"
)

func getProjectFromAPIKey(c *client.RancherClient) (*client.Project, error) {
	projects, err := c.Project.List(&client.ListOpts{})
	if err != nil || len(projects.Data) == 0 {
		return nil, err
	}
	return &projects.Data[0], nil
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
