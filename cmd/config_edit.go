package cmd

import (
	"os"

	"github.com/urfave/cli/v2"
	"github.com/vinaycharlie01/k3sctl/action"
)

func configEditCommand() *cli.Command {
	return &cli.Command{
		Name:  "edit",
		Usage: "Edit K3s configuration on the leader node",
		Flags: []cli.Flag{
			configFlag(),
		},
		Action: func(ctx *cli.Context) error {
			cfg, err := loadConfig(ctx.String("config"))
			if err != nil {
				return err
			}

			a := action.ConfigEdit{
				Config: cfg,
				Stdout: os.Stdout,
				Stderr: os.Stderr,
				Stdin:  os.Stdin,
			}

			return a.Run(ctx.Context)
		},
	}
}
