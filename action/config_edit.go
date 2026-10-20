package action

import (
	"context"
	"fmt"
	"io"
	"os"
	osexec "os/exec"
	"strings"

	"github.com/vinaycharlie01/k3sctl/phase"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
)

type ConfigEdit struct {
	Config *v1beta1.Cluster
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
}

func (c ConfigEdit) Run(ctx context.Context) error {
	m := &phase.Manager{}
	m.AddPhase(
		&phase.Connect{},
		&phase.DetectOS{},
		&phase.GatherFacts{},
		&editConfigPhase{
			Stdout: c.Stdout,
			Stderr: c.Stderr,
			Stdin:  c.Stdin,
		},
		&phase.Disconnect{},
	)

	return m.Run(ctx, c.Config)
}

type editConfigPhase struct {
	phase.GenericPhase
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
}

func (p *editConfigPhase) Title() string {
	return "Edit K3s configuration"
}

func (p *editConfigPhase) Run(ctx context.Context) error {
	leader := p.Config.Spec.Hosts.Leader()
	if leader == nil {
		return fmt.Errorf("no leader found in config")
	}

	configPath := leader.K3sConfigDir()
	oldCfg, err := leader.Sudo().ExecOutput(fmt.Sprintf("cat %s 2>/dev/null || echo ''", shellQuote(configPath)))
	if err != nil {
		return fmt.Errorf("%s: read config: %w", leader, err)
	}

	tmpFile, err := os.CreateTemp("", "k3s-config.*.yaml")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(oldCfg); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	editor, err := shellEditor()
	if err != nil {
		return err
	}

	cmd := osexec.CommandContext(ctx, editor, tmpFile.Name())
	cmd.Stdin = p.Stdin
	cmd.Stdout = p.Stdout
	cmd.Stderr = p.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor (%s) failed: %w", editor, err)
	}

	newCfgBytes, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return err
	}
	newCfg := string(newCfgBytes)

	if newCfg == oldCfg {
		return fmt.Errorf("configuration was not changed, aborting")
	}

	sudoFS := leader.Sudo().FS()
	if err := sudoFS.WriteFile(configPath, newCfgBytes, 0600); err != nil {
		return fmt.Errorf("%s: write config: %w", leader, err)
	}

	fmt.Fprintln(p.Stderr, "Configuration updated. Restart K3s for changes to take effect.")
	return nil
}

func shellEditor() (string, error) {
	if v := os.Getenv("VISUAL"); v != "" {
		return v, nil
	}
	if v := os.Getenv("EDITOR"); v != "" {
		return v, nil
	}
	if path, err := osexec.LookPath("vi"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("no editor found (set $VISUAL or $EDITOR)")
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
