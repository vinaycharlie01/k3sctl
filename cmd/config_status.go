package cmd

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/vinaycharlie01/k3sctl/action"
)

func configStatusCommand() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "Show K3s cluster status",
		Flags: []cli.Flag{
			configFlag(),
			concurrencyFlag(),
			&cli.BoolFlag{
				Name:    "deep",
				Aliases: []string{"d"},
				Usage:   "Run deep health checks (API server, etcd, scheduler, controller-manager, node readiness)",
			},
		},
		Action: func(ctx *cli.Context) error {
			cfg, err := loadConfig(ctx.String("config"))
			if err != nil {
				return err
			}

			m, _ := action.ConfigStatus(action.ConfigStatusOptions{
				Writer: os.Stdout,
				Deep:   ctx.Bool("deep"),
			})

			m.Concurrency = ctx.Int("concurrency")

			if err := m.Run(ctx.Context, cfg); err != nil {
				return err
			}

			log.Info("==> Status complete")
			return nil
		},
	}
}
