package main

import (
	"os"

	"github.com/codegangsta/cli"
	"github.com/rancher/secrets-bridge/cmd"
)

func main() {
	app := cli.NewApp()
	app.Name = "secrets-bridge"
	app.Usage = "Bridge containers with a secret"

	app.Commands = []cli.Command{
		cmd.ServerCommand(),
		cmd.AgentCommand(),
	}

	app.Run(os.Args)
}
