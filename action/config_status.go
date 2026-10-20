package action

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/vinaycharlie01/k3sctl/phase"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type ConfigStatusOptions struct {
	Writer io.Writer
}

func ConfigStatus(opts ConfigStatusOptions) *phase.Manager {
	m := &phase.Manager{}
	m.AddPhase(
		&phase.Connect{},
		&phase.DetectOS{},
		&phase.GatherFacts{},
		&phase.GatherK3sFacts{},
		&printStatusPhase{Writer: opts.Writer},
		&phase.Disconnect{},
	)
	return m
}

type printStatusPhase struct {
	phase.GenericPhase
	Writer io.Writer
}

func (p *printStatusPhase) Title() string {
	return "Print cluster status"
}

func (p *printStatusPhase) Run(_ context.Context) error {
	PrintStatus(p.Writer, p.Config)
	return nil
}

func PrintStatus(w io.Writer, cfg *v1beta1.Cluster) {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "HOST\tROLE\tSTATUS\tK3S VERSION")
	fmt.Fprintln(tw, "----\t----\t------\t-----------")

	for _, h := range cfg.Spec.Hosts {
		status := hostStatus(h)
		version := h.Metadata.K3sInstalledVersion
		if version == "" {
			version = "-"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", h, h.Role, status, version)
	}
	_ = tw.Flush()
}

func hostStatus(h *cluster.Host) string {
	if !h.Metadata.K3sServiceInstalled {
		return "not installed"
	}
	if h.Metadata.K3sServiceRunning {
		return "running"
	}
	return "stopped"
}
