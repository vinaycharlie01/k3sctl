package cmd

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/vinaycharlie01/k3sctl/action"
)

func configDiffCommand() *cli.Command {
	return &cli.Command{
		Name:  "diff",
		Usage: "Show differences between desired configuration and actual cluster state",
		Flags: []cli.Flag{
			configFlag(),
			concurrencyFlag(),
		},
		Action: func(ctx *cli.Context) error {
			cfg, err := loadConfig(ctx.String("config"))
			if err != nil {
				return err
			}

			m := action.ConfigDiff(action.ConfigDiffOptions{
				Writer: os.Stdout,
			})

			m.Concurrency = ctx.Int("concurrency")

			if err := m.Run(ctx.Context, cfg); err != nil {
				return err
			}

			log.Info("==> Config diff complete")
			return nil
		},
	}
}
