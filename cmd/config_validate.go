package cmd

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func configValidateCommand() *cli.Command {
	return &cli.Command{
		Name:  "validate",
		Usage: "Validate a cluster configuration file",
		Flags: []cli.Flag{
			configFlag(),
		},
		Action: func(ctx *cli.Context) error {
			_, err := loadConfig(ctx.String("config"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "validation failed: %v\n", err)
				return err
			}
			log.Info("configuration is valid")
			return nil
		},
	}
}
