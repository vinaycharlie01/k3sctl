package cmd

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/vinaycharlie01/k3sctl/action"
)

func backupCommand() *cli.Command {
	return &cli.Command{
		Name:  "backup",
		Usage: "Backup K3s cluster etcd snapshot",
		Flags: []cli.Flag{
			configFlag(),
			concurrencyFlag(),
			&cli.StringFlag{
				Name:    "out",
				Aliases: []string{"o"},
				Usage:   "Output `FILE` for the backup",
				Value:   "backup.db",
			},
		},
		Action: func(ctx *cli.Context) error {
			displayLogo()

			cfg, err := loadConfig(ctx.String("config"))
			if err != nil {
				return err
			}

			outFile, err := os.Create(ctx.String("out"))
			if err != nil {
				return fmt.Errorf("create backup file: %w", err)
			}
			defer func() { _ = outFile.Close() }()

			m := action.Backup(action.BackupOptions{
				Out: outFile,
			})
			m.Concurrency = ctx.Int("concurrency")

			if err := m.Run(ctx.Context, cfg); err != nil {
				return err
			}

			log.Infof("==> Backup saved to %s", ctx.String("out"))
			return nil
		},
	}
}
