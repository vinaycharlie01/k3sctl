package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/vinaycharlie01/k3sctl/action"
)

func etcdSnapshotCommand() *cli.Command {
	return &cli.Command{
		Name:  "etcd-snapshot",
		Usage: "Manage etcd snapshots",
		Subcommands: []*cli.Command{
			etcdSnapshotListCommand(),
			etcdSnapshotDeleteCommand(),
		},
	}
}

func etcdSnapshotListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List etcd snapshots on the server",
		Flags: []cli.Flag{
			configFlag(),
			concurrencyFlag(),
		},
		Action: func(ctx *cli.Context) error {
			displayLogo()

			cfg, err := loadConfig(ctx.String("config"))
			if err != nil {
				return err
			}

			m, listPhase := action.EtcdSnapshotList()
			m.Concurrency = ctx.Int("concurrency")

			if err := m.Run(ctx.Context, cfg); err != nil {
				return err
			}

			if len(listPhase.Snapshots) == 0 {
				log.Info("No snapshots found")
				return nil
			}

			fmt.Printf("%-50s %-12s %s\n", "NAME", "SIZE", "CREATED")
			for _, s := range listPhase.Snapshots {
				fmt.Printf("%-50s %-12s %s\n", s.Name, s.Size, s.Created)
			}

			return nil
		},
	}
}

func etcdSnapshotDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete an etcd snapshot",
		ArgsUsage: "SNAPSHOT_NAME",
		Flags: []cli.Flag{
			configFlag(),
			concurrencyFlag(),
		},
		Action: func(ctx *cli.Context) error {
			name := ctx.Args().First()
			if name == "" {
				return fmt.Errorf("snapshot name is required")
			}

			displayLogo()

			cfg, err := loadConfig(ctx.String("config"))
			if err != nil {
				return err
			}

			m := action.EtcdSnapshotDelete(name)
			m.Concurrency = ctx.Int("concurrency")

			if err := m.Run(ctx.Context, cfg); err != nil {
				return err
			}

			log.Infof("==> Snapshot %s deleted", name)
			return nil
		},
	}
}
