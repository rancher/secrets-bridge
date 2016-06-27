package cmd

import (
	"github.com/rancher/secrets-bridge/bridge"
	"github.com/urfave/cli"
)

func ServerCommand() cli.Command {
	return cli.Command{
		Name:   "server",
		Usage:  "Provides a Secrets endpoint for verification and credential creation",
		Action: bridge.StartServer,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "vault-url",
				Usage: "URL to Vault server. http://127.0.0.1:9000",
			},
			cli.StringFlag{
				Name:   "vault-token",
				Usage:  "CubbyHole Vault Token to use to communicate with Vault",
				EnvVar: "VAULT_TOKEN",
			},
			cli.StringFlag{
				Name:   "vault-cubbypath",
				Usage:  "CubbyHole path to get Vault Token",
				EnvVar: "VAULT_CUBBYPATH",
			},
			cli.StringFlag{
				Name:   "rancher-url",
				Usage:  "Rancher API endpoint to verify",
				EnvVar: "CATTLE_URL",
			},
			cli.StringFlag{
				Name:   "rancher-secret",
				Usage:  "Rancher API secret key",
				EnvVar: "CATTLE_SECRET_KEY",
			},
			cli.StringFlag{
				Name:   "rancher-access",
				Usage:  "Rancher API access key",
				EnvVar: "CATTLE_ACCESS_KEY",
			},
		},
	}
}
