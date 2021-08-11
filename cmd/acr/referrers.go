package main

import "github.com/urfave/cli/v2"

const (
	referrersCountStr = "referrers"
)

var (
	checkReferrersflags = []cli.Flag{
		&cli.IntFlag{
			Name:  referrersCountStr,
			Usage: "number of referrers to create",
			Value: 1,
		},
	}

	referrersCommand = &cli.Command{
		Name:      "check-referrers",
		Usage:     "check referrers data path (push, pull) based on https://github.com/opencontainers/artifacts/pull/29",
		ArgsUsage: "<login-server>",
		Flags:     append(commonFlags, checkReferrersflags...),
		Action:    runCheckReferrers,
	}
)

func runCheckReferrers(ctx *cli.Context) (err error) {
	proxy, err := proxy(ctx)
	if err != nil {
		return err
	}

	err = proxy.Ping()
	if err != nil {
		return err
	}

	err = proxy.CheckReferrers(ctx.Int(referrersCountStr))
	if err != nil {
		return err
	}

	return nil
}
