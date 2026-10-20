package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/vinaycharlie01/k3sctl/action"
)

func resetCommand() *cli.Command {
	return &cli.Command{
		Name:  "reset",
		Usage: "Reset a K3s cluster (remove K3s from all hosts)",
		Flags: []cli.Flag{
			configFlag(),
			concurrencyFlag(),
			dryRunFlag(),
			&cli.BoolFlag{
				Name:  "no-drain",
				Usage: "Don't drain nodes before reset",
			},
			&cli.BoolFlag{
				Name:  "no-delete",
				Usage: "Don't delete nodes from cluster before reset",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Skip confirmation prompt",
			},
		},
		Action: func(ctx *cli.Context) error {
			displayLogo()

			cfg, err := loadConfig(ctx.String("config"))
			if err != nil {
				return err
			}

			if !ctx.Bool("force") {
				fmt.Println("This will remove K3s from all configured hosts.")
				fmt.Print("Are you sure? (y/N): ")
				var answer string
				_, _ = fmt.Scanln(&answer)
				if answer != "y" && answer != "Y" {
					return fmt.Errorf("reset cancelled")
				}
			}

			for _, h := range cfg.Spec.Hosts {
				h.Reset = true
			}

			m := action.Reset(action.ResetOptions{
				NoDrain:  ctx.Bool("no-drain"),
				NoDelete: ctx.Bool("no-delete"),
			})
			m.DryRun = ctx.Bool("dry-run")
			m.Concurrency = ctx.Int("concurrency")

			if err := m.Run(ctx.Context, cfg); err != nil {
				return err
			}

			log.Info("==> Reset complete")
			return nil
		},
	}
}
