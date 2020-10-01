package main

import "github.com/urfave/cli/v2"

var checkHealthCommand = &cli.Command{
	Name:      "check-health",
	Usage:     "check health of registry endpoints",
	ArgsUsage: "<login-server>",
	Flags:     commonFlags,
	Action:    runCheckHealth,
}

func runCheckHealth(ctx *cli.Context) (err error) {
	proxy, err := proxy(ctx)
	if err != nil {
		return err
	}

	err = proxy.Ping()
	if err != nil {
		return err
	}

	err = proxy.CheckHealth()
	if err != nil {
		return err
	}

	return nil
}
