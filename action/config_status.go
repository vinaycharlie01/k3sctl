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
	Deep   bool
}

func ConfigStatus(opts ConfigStatusOptions) (*phase.Manager, *phase.HealthCheck) {
	healthPhase := &phase.HealthCheck{}
	m := &phase.Manager{}
	phases := []phase.Phase{
		&phase.Connect{},
		&phase.DetectOS{},
		&phase.GatherFacts{},
		&phase.GatherK3sFacts{},
		&printStatusPhase{Writer: opts.Writer, Deep: opts.Deep, Health: healthPhase},
	}
	if opts.Deep {
		// Insert health check before print
		phases = []phase.Phase{
			&phase.Connect{},
			&phase.DetectOS{},
			&phase.GatherFacts{},
			&phase.GatherK3sFacts{},
			healthPhase,
			&printStatusPhase{Writer: opts.Writer, Deep: opts.Deep, Health: healthPhase},
		}
	}
	phases = append(phases, &phase.Disconnect{})
	m.AddPhase(phases...)
	return m, healthPhase
}

type printStatusPhase struct {
	phase.GenericPhase
	Writer io.Writer
	Deep   bool
	Health *phase.HealthCheck
}

func (p *printStatusPhase) Title() string {
	return "Print cluster status"
}

func (p *printStatusPhase) Run(_ context.Context) error {
	if p.Deep && p.Health != nil && len(p.Health.Results) > 0 {
		PrintDeepStatus(p.Writer, p.Config, p.Health.Results)
	} else {
		PrintStatus(p.Writer, p.Config)
	}
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

func PrintDeepStatus(w io.Writer, cfg *v1beta1.Cluster, results []phase.HealthInfo) {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "HOST\tROLE\tSTATUS\tVERSION\tAPI\tETCD\tSCHEDULER\tCTRL-MGR\tNODE\tPODS")
	fmt.Fprintln(tw, "----\t----\t------\t-------\t---\t----\t---------\t--------\t----\t----")

	lookup := make(map[string]phase.HealthInfo, len(results))
	for _, r := range results {
		lookup[r.Host] = r
	}

	for _, h := range cfg.Spec.Hosts {
		info, ok := lookup[h.String()]
		if !ok {
			fmt.Fprintf(tw, "%s\t%s\t-\t-\t-\t-\t-\t-\t-\t-\n", h, h.Role)
			continue
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			info.Host, info.Role, info.K3sStatus, info.K3sVersion,
			info.APIServer, info.Etcd, info.Scheduler, info.ControllerMgr,
			info.NodeReady, info.PodCount)
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
