package cmd

import (
	"fmt"
	"os"
	"text/template"

	"github.com/urfave/cli/v2"
)

const initTemplate = `apiVersion: k3sctl.k3sproject.io/v1beta1
kind: Cluster
metadata:
  name: {{ .Name }}
spec:
  hosts:
  - role: server
    ssh:
      address: {{ .Address }}
      user: {{ .User }}
      port: {{ .Port }}
      keyPath: {{ .KeyPath }}
{{- if .Agent }}
  - role: agent
    ssh:
      address: {{ .AgentAddress }}
      user: {{ .User }}
      port: {{ .Port }}
      keyPath: {{ .KeyPath }}
{{- end }}
  k3s:
    version: "{{ .Version }}"
`

type initData struct {
	Name         string
	Address      string
	AgentAddress string
	User         string
	Port         int
	KeyPath      string
	Version      string
	Agent        bool
}

func initCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Generate a cluster configuration template",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "name",
				Usage: "Cluster name",
				Value: "k3s-cluster",
			},
			&cli.StringFlag{
				Name:  "address",
				Usage: "Server host address",
				Value: "10.0.0.1",
			},
			&cli.StringFlag{
				Name:  "user",
				Usage: "SSH user",
				Value: "root",
			},
			&cli.IntFlag{
				Name:  "port",
				Usage: "SSH port",
				Value: 22,
			},
			&cli.StringFlag{
				Name:  "key-path",
				Usage: "SSH key path",
				Value: "~/.ssh/id_rsa",
			},
			&cli.StringFlag{
				Name:  "k3s-version",
				Usage: "K3s version",
			},
			&cli.BoolFlag{
				Name:  "agent",
				Usage: "Include an agent host template",
			},
			&cli.StringFlag{
				Name:  "agent-address",
				Usage: "Agent host address",
				Value: "10.0.0.2",
			},
		},
		Action: func(ctx *cli.Context) error {
			data := initData{
				Name:         ctx.String("name"),
				Address:      ctx.String("address"),
				User:         ctx.String("user"),
				Port:         ctx.Int("port"),
				KeyPath:      ctx.String("key-path"),
				Version:      ctx.String("k3s-version"),
				Agent:        ctx.Bool("agent"),
				AgentAddress: ctx.String("agent-address"),
			}

			tmpl, err := template.New("init").Parse(initTemplate)
			if err != nil {
				return fmt.Errorf("parse template: %w", err)
			}

			return tmpl.Execute(os.Stdout, data)
		},
	}
}
