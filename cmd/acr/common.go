package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/aviral26/acr-checkhealth/pkg/registry"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

// Common flag names
const (
	insecureStr     = "insecure"
	userNameStr     = "username"
	passwordStr     = "password"
	dataEndpointStr = "dataendpoint"
	traceStr        = "trace"
)

// Common cli flags
var (
	insecureFlag = &cli.BoolFlag{
		Name:  insecureStr,
		Usage: "enable remote access over HTTP",
	}

	usernameFlag = &cli.StringFlag{
		Name:    userNameStr,
		Aliases: []string{"u"},
		Usage:   "login username",
	}

	passwordFlag = &cli.StringFlag{
		Name:    passwordStr,
		Aliases: []string{"p"},
		Usage:   "login password",
	}

	dataEndpointFlag = &cli.StringFlag{
		Name:    dataEndpointStr,
		Aliases: []string{"d"},
		Usage:   "endpoint for data download",
	}

	traceFlag = &cli.BoolFlag{
		Name:  traceStr,
		Usage: "print trace logs",
	}
)

var (
	logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
)

// proxy creates an new proxy instance from context specific arguments and flags.
func proxy(ctx *cli.Context) (p *registry.Proxy, err error) {
	var loginServer, dataEndpoint string

	if loginServer, dataEndpoint, err = resolveAll(ctx); err != nil {
		return nil, err
	}

	if ctx.Bool(traceStr) {
		logger = logger.With().Logger().Level(zerolog.TraceLevel)
	} else {
		logger = logger.With().Logger().Level(zerolog.InfoLevel)
	}

	return registry.NewProxy(http.DefaultTransport,
		&registry.Options{
			LoginServer:  loginServer,
			Username:     ctx.String(userNameStr),
			Password:     ctx.String(passwordStr),
			DataEndpoint: dataEndpoint,
			Insecure:     ctx.Bool(insecureStr),
		},
		logger)
}

// resolveAll attempts to resolve the endpoints specified in the
func resolveAll(ctx *cli.Context) (loginServer, dataEndpoint string, err error) {
	hostnames := []string{}

	if loginServer = ctx.Args().First(); loginServer == "" {
		return loginServer, dataEndpoint, errors.New("login server name required")
	}

	hostnames = append(hostnames, loginServer)

	if dataEndpoint = ctx.String(dataEndpointStr); dataEndpoint != "" {
		hostnames = append(hostnames, dataEndpoint)
	}

	for _, hostname := range hostnames {
		if err := resolve(hostname); err != nil {
			return loginServer, dataEndpoint, err
		}
	}

	return loginServer, dataEndpoint, nil
}

// resolve ..
// dig +short hostname
func resolve(hostname string) error {
	if hostname == "" {
		return errors.New("hostname required")
	}

	path := []string{}

	cur := hostname
	path = append(path, cur)
	for {
		cname, err := net.LookupCNAME(cur)
		if err != nil {
			return err
		}
		if cname == cur {
			// No more aliases.
			break
		}
		path = append(path, cname)
		cur = cname
	}

	ip, err := net.LookupIP(cur)
	if err != nil {
		return err
	}
	path = append(path, ip[0].String())

	logger.Info().Msg(fmt.Sprintf("DNS:  %v", strings.Join(path, " -> ")))
	return nil
}
