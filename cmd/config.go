package cmd

import "github.com/urfave/cli/v2"

func configCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Configuration related sub-commands",
		Subcommands: []*cli.Command{
			configEditCommand(),
			configStatusCommand(),
		},
	}
}
