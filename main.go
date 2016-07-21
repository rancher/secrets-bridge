package main

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/secrets-bridge/cmd"
	"github.com/urfave/cli"
)

func beforeApp(c *cli.Context) error {
	if c.GlobalBool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
	}
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "secrets-bridge"
	app.Usage = "Bridge containers with a secret"
	app.Before = beforeApp
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name: "debug,d",
		},
	}

	app.Commands = []cli.Command{
		cmd.ServerCommand(),
		cmd.AgentCommand(),
	}

	app.Run(os.Args)
}
