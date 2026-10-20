package cmd

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func completionCommand() *cli.Command {
	return &cli.Command{
		Name:  "completion",
		Usage: "Generate shell completion scripts",
		Subcommands: []*cli.Command{
			{
				Name:  "bash",
				Usage: "Generate bash completion script",
				Action: func(ctx *cli.Context) error {
					script := `#! /bin/bash
_k3sctl_completions() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local commands="apply reset kubeconfig backup init version completion"
    COMPREPLY=( $(compgen -W "${commands}" -- "${cur}") )
}
complete -F _k3sctl_completions k3sctl`
					fmt.Fprintln(os.Stdout, script)
					return nil
				},
			},
			{
				Name:  "zsh",
				Usage: "Generate zsh completion script",
				Action: func(ctx *cli.Context) error {
					script := `#compdef k3sctl
_k3sctl() {
    local -a commands
    commands=(
        'apply:Apply a K3s cluster configuration'
        'reset:Reset a K3s cluster'
        'kubeconfig:Get the admin kubeconfig'
        'backup:Backup K3s cluster etcd snapshot'
        'init:Generate a cluster configuration template'
        'version:Print version information'
        'completion:Generate shell completion scripts'
    )
    _describe 'command' commands
}
_k3sctl`
					fmt.Fprintln(os.Stdout, script)
					return nil
				},
			},
		},
	}
}
