package cmd

import (
	"fmt"
	"sort"

	"github.com/urfave/cli/v2"
	"github.com/vinaycharlie01/k3sctl/action"
)

func certificateCommand() *cli.Command {
	return &cli.Command{
		Name:    "certificate",
		Aliases: []string{"cert"},
		Usage:   "Manage K3s certificates",
		Subcommands: []*cli.Command{
			certificateCheckCommand(),
		},
	}
}

func certificateCheckCommand() *cli.Command {
	return &cli.Command{
		Name:  "check",
		Usage: "Check certificate expiry dates",
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

			m, checkPhase := action.CertificateCheck()
			m.Concurrency = ctx.Int("concurrency")

			if err := m.Run(ctx.Context, cfg); err != nil {
				return err
			}

			results := checkPhase.Results
			if len(results) == 0 {
				fmt.Println("No certificates found.")
				return nil
			}

			sort.Slice(results, func(i, j int) bool {
				if results[i].Host != results[j].Host {
					return results[i].Host < results[j].Host
				}
				return results[i].Name < results[j].Name
			})

			fmt.Printf("%-20s %-30s %-14s %-10s %s\n", "HOST", "CERTIFICATE", "EXPIRES", "DAYS LEFT", "STATUS")
			fmt.Printf("%-20s %-30s %-14s %-10s %s\n", "----", "-----------", "-------", "---------", "------")

			for _, c := range results {
				if c.Error != "" {
					fmt.Printf("%-20s %-30s %-14s %-10s %s\n", c.Host, c.Name, "error", "-", c.Error)
					continue
				}

				status := "OK"
				if c.DaysLeft < 0 {
					status = "EXPIRED"
				} else if c.DaysLeft < 30 {
					status = "EXPIRING"
				}

				fmt.Printf("%-20s %-30s %-14s %-10d %s\n", c.Host, c.Name, c.NotAfter, c.DaysLeft, status)
			}

			return nil
		},
	}
}
