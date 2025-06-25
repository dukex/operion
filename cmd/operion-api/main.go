package main

import (
	"context"
	"os"

	"github.com/go-playground/validator/v10"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v3"
)

var validate *validator.Validate

func main() {
	cmd := &cli.Command{
		Name:                  "operion-api",
		Usage:                 "Create and manage workflows",
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			RunAPICommand(),
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
