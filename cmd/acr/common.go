package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

const (
	insecureStr     = "insecure"
	userNameStr     = "username"
	passwordStr     = "password"
	dataEndpointStr = "dataendpoint"
)

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
)

var (
	logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
)

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
