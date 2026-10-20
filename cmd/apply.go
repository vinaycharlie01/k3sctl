package cmd

import (
	"fmt"
	"io"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/vinaycharlie01/k3sctl/action"
)

func applyCommand() *cli.Command {
	return &cli.Command{
		Name:  "apply",
		Usage: "Apply a K3s cluster configuration",
		Flags: []cli.Flag{
			configFlag(),
			concurrencyFlag(),
			concurrentUploadsFlag(),
			dryRunFlag(),
			&cli.BoolFlag{
				Name:  "no-wait",
				Usage: "Don't wait for the cluster to be ready",
			},
			&cli.BoolFlag{
				Name:  "no-drain",
				Usage: "Don't drain worker nodes during upgrade",
			},
			&cli.StringFlag{
				Name:  "restore-from",
				Usage: "Path to etcd snapshot to restore from",
			},
			&cli.StringFlag{
				Name:  "kubeconfig-out",
				Usage: "Write admin kubeconfig to `FILE`",
			},
			&cli.StringFlag{
				Name:  "kubeconfig-api-address",
				Usage: "Override the API address in the kubeconfig",
			},
			&cli.StringFlag{
				Name:  "kubeconfig-user",
				Usage: "Override the user in the kubeconfig",
			},
			&cli.StringFlag{
				Name:  "kubeconfig-cluster",
				Usage: "Override the cluster name in the kubeconfig",
			},
			&cli.BoolFlag{
				Name:  "disable-downgrade-check",
				Usage: "Allow downgrading K3s versions",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Skip confirmation prompts",
			},
		},
		Action: func(ctx *cli.Context) error {
			displayLogo()

			cfg, err := loadConfig(ctx.String("config"))
			if err != nil {
				return err
			}

			// Keep this typed as io.Writer: a nil *os.File assigned to the
			// interface would be non-nil, so Apply would add the kubeconfig
			// phase and writing would fail on a nil file handle.
			var kubeconfigOut io.Writer
			if path := ctx.String("kubeconfig-out"); path != "" {
				f, err := os.Create(path)
				if err != nil {
					return fmt.Errorf("create kubeconfig file: %w", err)
				}
				defer func() { _ = f.Close() }()
				kubeconfigOut = f
			}

			m := action.Apply(action.ApplyOptions{
				DisableDowngradeCheck: ctx.Bool("disable-downgrade-check"),
				NoWait:                ctx.Bool("no-wait"),
				NoDrain:               ctx.Bool("no-drain"),
				RestoreFrom:           ctx.String("restore-from"),
				KubeconfigOut:         kubeconfigOut,
				KubeconfigAPIAddress:  ctx.String("kubeconfig-api-address"),
				KubeconfigUser:        ctx.String("kubeconfig-user"),
				KubeconfigCluster:     ctx.String("kubeconfig-cluster"),
			})

			m.DryRun = ctx.Bool("dry-run")
			m.Concurrency = ctx.Int("concurrency")
			m.ConcurrentUploads = ctx.Int("concurrent-uploads")

			if err := m.Run(ctx.Context, cfg); err != nil {
				return err
			}

			log.Info("==> Apply complete")

			if kubeconfigOut == nil {
				log.Info("To get the kubeconfig, run: k3sctl kubeconfig")
			}

			return nil
		},
	}
}
