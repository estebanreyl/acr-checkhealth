package main

import (
	"fmt"
	"net/http"

	"github.com/aviral26/acr/runner/pkg/registry"
	"github.com/urfave/cli/v2"
)

var pingCommand = &cli.Command{
	Name:      "ping",
	Usage:     "ping registry endpoints",
	ArgsUsage: "[<login-server>]",
	Flags: []cli.Flag{
		usernameFlag,
		passwordFlag,
		dataEndpointFlag,
		insecureFlag,
	},
	Action: runPing,
}

func runPing(ctx *cli.Context) error {
	proxy, err := registry.NewProxy(http.DefaultTransport,
		&registry.Options{
			LoginServer: ctx.Args().First(),
			Username:    ctx.String(userNameStr),
			Password:    ctx.String(passwordStr),
		})
	if err != nil {
		return err
	}

	defer func() {
		fmt.Print(proxy.Logs())
	}()

	err = proxy.Ping()
	if err != nil {
		return err
	}

	return nil
}
