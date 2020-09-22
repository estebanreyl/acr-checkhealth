package main

import (
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
