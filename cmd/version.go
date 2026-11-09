package cmd

import (
	"fmt"

	"github.com/nirantarasi/k3sctl/integration/github"
	"github.com/nirantarasi/k3sctl/version"
	"github.com/urfave/cli/v2"
)

func versionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Print k3sctl version information",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "k3s",
				Usage: "Also print the latest available K3s version",
			},
			&cli.BoolFlag{
				Name:  "pre",
				Usage: "Include pre-release versions",
			},
		},
		Action: func(ctx *cli.Context) error {
			fmt.Printf("k3sctl version: %s\n", version.Version)
			if version.GitCommit != "" {
				fmt.Printf("git commit: %s\n", version.GitCommit)
			}

			if ctx.Bool("k3s") {
				v, err := github.LatestK3sVersion(ctx.Context, false)
				if err != nil {
					return fmt.Errorf("fetch latest K3s version: %w", err)
				}
				fmt.Printf("latest k3s version: %s\n", v)
			}

			return nil
		},
	}
}
