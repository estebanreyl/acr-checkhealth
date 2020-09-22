package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "runner",
		Usage:   "ACR Runner - evaluate if an Azure Container Registry is healthy",
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
		log.Fatal(err)
	}
}
