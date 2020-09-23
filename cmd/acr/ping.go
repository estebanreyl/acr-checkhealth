package main

import (
	"net/http"

	"github.com/aviral26/acr/conformance/pkg/registry"
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
	loginServer := ""
	dataEndpoint := ""
	if loginServer, dataEndpoint, err = resolveAll(ctx); err != nil {
		return err
	}

	proxy, err := registry.NewProxy(http.DefaultTransport,
		&registry.Options{
			LoginServer:  loginServer,
			Username:     ctx.String(userNameStr),
			Password:     ctx.String(passwordStr),
			DataEndpoint: dataEndpoint,
		},
		logger)
	if err != nil {
		return err
	}

	err = proxy.Ping()
	if err != nil {
		return err
	}

	return nil
}
