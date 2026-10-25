package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/vinaycharlie01/k3sctl/action"
)

func restoreCommand() *cli.Command {
	return &cli.Command{
		Name:  "restore",
		Usage: "Restore K3s cluster from an etcd snapshot",
		Flags: []cli.Flag{
			configFlag(),
			concurrencyFlag(),
			&cli.StringFlag{
				Name:     "from",
				Aliases:  []string{"f"},
				Usage:    "Path to etcd snapshot `FILE` to restore from",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Skip confirmation prompt",
			},
		},
		Action: func(ctx *cli.Context) error {
			displayLogo()

			snapshotPath := ctx.String("from")
			if _, err := os.Stat(snapshotPath); err != nil {
				return fmt.Errorf("snapshot file: %w", err)
			}

			cfg, err := loadConfig(ctx.String("config"))
			if err != nil {
				return err
			}

			if !ctx.Bool("force") {
				fmt.Fprintf(os.Stderr, "WARNING: This will stop K3s, reset etcd state, and restore from %q.\n", snapshotPath)
				fmt.Fprintf(os.Stderr, "Cluster: %s\n", cfg.Metadata.Name)
				fmt.Fprint(os.Stderr, "Continue? [y/N]: ")

				reader := bufio.NewReader(os.Stdin)
				answer, _ := reader.ReadString('\n')
				if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(answer)), "y") {
					return fmt.Errorf("restore aborted by user")
				}
			}

			m := action.Restore(action.RestoreOptions{
				RestoreFrom: snapshotPath,
			})
			m.Concurrency = ctx.Int("concurrency")

			if err := m.Run(ctx.Context, cfg); err != nil {
				return err
			}

			log.Infof("==> Cluster restored from %s", snapshotPath)
			return nil
		},
	}
}
