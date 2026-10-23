package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli/v2"
	"github.com/vinaycharlie01/k3sctl/action"
	"github.com/vinaycharlie01/k3sctl/phase"
)

func tokenCommand() *cli.Command {
	return &cli.Command{
		Name:  "token",
		Usage: "Manage K3s cluster tokens",
		Subcommands: []*cli.Command{
			tokenListCommand(),
		},
	}
}

func tokenListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List tokens from the K3s cluster",
		Flags: []cli.Flag{
			configFlag(),
			concurrencyFlag(),
		},
		Action: func(ctx *cli.Context) error {
			displayLogo()

			cfg, err := loadConfig(ctx.String("config"))
			if err != nil {
				return err
			}

			m, listPhase := action.TokenList()
			m.Concurrency = ctx.Int("concurrency")

			if err := m.Run(ctx.Context, cfg); err != nil {
				return err
			}

			if len(listPhase.Tokens) == 0 {
				fmt.Fprintln(os.Stderr, "No tokens found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "HOST\tTYPE\tTOKEN")
			for _, t := range listPhase.Tokens {
				fmt.Fprintf(w, "%s\t%s\t%s\n", t.Host, t.Type, maskToken(t.Token))
			}
			return w.Flush()
		},
	}
}

func maskToken(token string) string {
	if len(token) > 20 {
		return token[:8] + "..." + token[len(token)-4:]
	}
	if len(token) > 4 {
		return token[:4] + "..."
	}
	return token
}

// TokenInfo is re-exported for callers that need the type without importing the phase package.
type TokenInfo = phase.TokenInfo
