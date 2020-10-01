package main

import "github.com/urfave/cli/v2"

var referrersCommand = &cli.Command{
	Name:      "check-referrers",
	Usage:     "check referrers data path (push, pull) based on https://github.com/opencontainers/artifacts/pull/29",
	ArgsUsage: "<login-server>",
	Flags:     commonFlags,
	Action:    runCheckReferrers,
}

func runCheckReferrers(ctx *cli.Context) (err error) {
	proxy, err := proxy(ctx)
	if err != nil {
		return err
	}

	err = proxy.Ping()
	if err != nil {
		return err
	}

	err = proxy.CheckReferrers()
	if err != nil {
		return err
	}

	return nil
}
