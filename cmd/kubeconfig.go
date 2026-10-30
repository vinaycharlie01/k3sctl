package cmd

import (
	"os"

	"log/slog"

	"github.com/urfave/cli/v2"
	"github.com/vinaycharlie01/k3sctl/action"
)

func kubeconfigCommand() *cli.Command {
	return &cli.Command{
		Name:  "kubeconfig",
		Usage: "Get the admin kubeconfig for a K3s cluster",
		Flags: []cli.Flag{
			configFlag(),
			&cli.StringFlag{
				Name:  "api-address",
				Usage: "Override the API address in the kubeconfig",
			},
			&cli.StringFlag{
				Name:  "user",
				Usage: "Override the user in the kubeconfig",
			},
			&cli.StringFlag{
				Name:  "cluster",
				Usage: "Override the cluster name in the kubeconfig",
			},
		},
		Action: func(ctx *cli.Context) error {
			cfg, err := loadConfig(ctx.String("config"))
			if err != nil {
				return err
			}

			m := action.Kubeconfig(action.KubeconfigOptions{
				Out:        os.Stdout,
				APIAddress: ctx.String("api-address"),
				User:       ctx.String("user"),
				Cluster:    ctx.String("cluster"),
			})

			if err := m.Run(ctx.Context, cfg); err != nil {
				return err
			}

			slog.Debug("kubeconfig retrieved")
			return nil
		},
	}
}
