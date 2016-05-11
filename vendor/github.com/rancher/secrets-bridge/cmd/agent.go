package cmd

import (
	"github.com/codegangsta/cli"
	"github.com/rancher/secrets-bridge/agent"
)

func AgentCommand() cli.Command {
	return cli.Command{
		Name:   "agent",
		Usage:  "Start listening agent on docker host",
		Action: agent.StartAgent,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "metadata-url",
				Value: "http://rancher-metadata/2015-12-19",
				Usage: "Sets the metadata variable",
			},
			cli.StringFlag{
				Name:  "bridge-url",
				Usage: "Secrets Bridge endpoint",
			},
		},
	}
}
