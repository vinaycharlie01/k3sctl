package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/vinaycharlie01/k3sctl/action"
)

func nodeCommand() *cli.Command {
	return &cli.Command{
		Name:  "node",
		Usage: "Manage cluster nodes",
		Subcommands: []*cli.Command{
			nodeRemoveCommand(),
		},
	}
}

func nodeRemoveCommand() *cli.Command {
	return &cli.Command{
		Name:      "remove",
		Usage:     "Remove a node from the cluster (drain, delete, uninstall)",
		ArgsUsage: "HOSTNAME_OR_ADDRESS",
		Flags: []cli.Flag{
			configFlag(),
			concurrencyFlag(),
			dryRunFlag(),
			&cli.BoolFlag{
				Name:  "no-drain",
				Usage: "Don't drain the node before removing",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Force removal even if the node is the leader",
			},
		},
		Action: func(ctx *cli.Context) error {
			target := ctx.Args().First()
			if target == "" {
				return fmt.Errorf("hostname or address is required")
			}

			displayLogo()

			cfg, err := loadConfig(ctx.String("config"))
			if err != nil {
				return err
			}

			m := action.NodeRemove(action.NodeRemoveOptions{
				Target:  target,
				NoDrain: ctx.Bool("no-drain"),
				Force:   ctx.Bool("force"),
			})
			m.DryRun = ctx.Bool("dry-run")
			m.Concurrency = ctx.Int("concurrency")

			if err := m.Run(ctx.Context, cfg); err != nil {
				return err
			}

			log.Infof("==> Node %s removed", target)
			return nil
		},
	}
}
