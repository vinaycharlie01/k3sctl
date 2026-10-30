package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"
	"github.com/vinaycharlie01/k3sctl/version"
)

func NewK3sctl() *cli.App {
	app := &cli.App{
		Name:    "k3sctl",
		Usage:   "K3s cluster lifecycle management tool",
		Version: version.Version,
		Flags:   GlobalFlags(),
		Before:  initLogging,
		Commands: []*cli.Command{
			versionCommand(),
			applyCommand(),
			kubeconfigCommand(),
			resetCommand(),
			restoreCommand(),
			configCommand(),
			initCommand(),
			completionCommand(),
		},
	}

	return app
}

func Run() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	app := NewK3sctl()
	if err := app.RunContext(ctx, os.Args); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
