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
		Usage:                 "Manage receivers, process trigger events, and dispatch workflows",
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
