package main

import (
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "runner",
		Usage:   "ACR Conformance - evaluate if registry APIs conform to specs",
		Version: "0.1.0",
		Authors: []*cli.Author{
			{
				Name: "Aviral Takkar",
			},
		},
		Commands: []*cli.Command{
			pingCommand,
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Fatal().Msg(err.Error())
	} else {
		logger.Info().Msg("success")
	}
}
