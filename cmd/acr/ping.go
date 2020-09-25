package main

import (
	"github.com/urfave/cli/v2"
)

var pingCommand = &cli.Command{
	Name:      "ping",
	Usage:     "ping registry endpoints",
	ArgsUsage: "<login-server>",
	Flags: []cli.Flag{
		usernameFlag,
		passwordFlag,
		dataEndpointFlag,
		insecureFlag,
	},
	Action: runPing,
}

func runPing(ctx *cli.Context) (err error) {
	proxy, err := proxy(ctx)
	if err != nil {
		return err
	}

	err = proxy.Ping()
	if err != nil {
		return err
	}

	return nil
}
