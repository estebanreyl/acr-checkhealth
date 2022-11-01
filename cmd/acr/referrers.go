package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

const (
	referrersCountStr    = "referrers"
	OciReferrers         = "Referrers_OCI_V1"
	OciManifestReferrers = "Referrers_OCI_Manifest"
	OrasReferrers        = "Referrers_ORAS_V1"
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
	fmt.Print("\n----------------------------------------------TEST START-----------------------------------------------\n")

	for _, version := range []string{OrasReferrers, OciManifestReferrers, OciReferrers} {
		fmt.Printf("\n------------------------%s-------------------------\n", version)
		fmt.Print("----ORDERED----\n")

		err = proxy.CheckReferrers(ctx.Int(referrersCountStr), version)
		if err != nil {
			fmt.Print(err)
		}

		fmt.Print("\n----OUT OF ORDER----\n")
		err = proxy.CheckReferrersOutOfOrder(ctx.Int(referrersCountStr), version)
		if err != nil {
			fmt.Print(err)
		}

	}

	return nil
}
