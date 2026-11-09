package cmd

import (
	"os"

	"log/slog"

	"github.com/nirantarasi/k3sctl/action"
	"github.com/urfave/cli/v2"
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
			applyLoggingConfig(cfg)

			m := action.ConfigDiff(action.ConfigDiffOptions{
				Writer: os.Stdout,
			})

			m.Concurrency = ctx.Int("concurrency")

			if err := m.Run(ctx.Context, cfg); err != nil {
				return err
			}

			slog.Info("==> Config diff complete")
			return nil
		},
	}
}
