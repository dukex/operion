package main

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:                  "operion-dispatcher",
		Usage:                 "Manage workflow triggers and publish trigger events",
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			NewRunCommand(),
			NewListCommand(),
			NewValidateCommand(),
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
