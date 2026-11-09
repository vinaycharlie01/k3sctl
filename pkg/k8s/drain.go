package k8s

import (
	"context"
	"fmt"
	"io"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/drain"
)

// DrainOptions mirrors the YAML-configurable drain settings from cluster.DrainOption.
type DrainOptions struct {
	Timeout          time.Duration
	GracePeriodSecs  int
	DeleteEmptyDir   bool
	IgnoreDaemonSets bool
	DisableEviction  bool
	Force            bool
}

func (c *client) CordonNode(ctx context.Context, nodeName string) error {
	return setNodeUnschedulable(ctx, c.cs, nodeName, true)
}

func (c *client) UncordonNode(ctx context.Context, nodeName string) error {
	return setNodeUnschedulable(ctx, c.cs, nodeName, false)
}

func setNodeUnschedulable(ctx context.Context, cs kubernetes.Interface, nodeName string, unschedulable bool) error {
	node, err := cs.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get node %s: %w", nodeName, err)
	}
	if node.Spec.Unschedulable == unschedulable {
		return nil
	}
	helper := &drain.Helper{Ctx: ctx, Client: cs}
	if unschedulable {
		return drain.RunCordonOrUncordon(helper, node, true)
	}
	return drain.RunCordonOrUncordon(helper, node, false)
}

func (c *client) DrainNode(ctx context.Context, nodeName string, opts DrainOptions) error {
	node, err := c.cs.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get node %s: %w", nodeName, err)
	}

	helper := &drain.Helper{
		Ctx:                             ctx,
		Client:                          c.cs,
		Force:                           opts.Force,
		IgnoreAllDaemonSets:             opts.IgnoreDaemonSets,
		DeleteEmptyDirData:              opts.DeleteEmptyDir,
		DisableEviction:                 opts.DisableEviction,
		Timeout:                         opts.Timeout,
		GracePeriodSeconds:              opts.GracePeriodSecs,
		Out:                             io.Discard,
		ErrOut:                          io.Discard,
		AdditionalFilters:               nil,
		SkipWaitForDeleteTimeoutSeconds: 0,
	}

	if err := drain.RunCordonOrUncordon(helper, node, true); err != nil {
		return fmt.Errorf("cordon %s before drain: %w", nodeName, err)
	}

	if err := drain.RunNodeDrain(helper, node.Name); err != nil {
		return fmt.Errorf("drain %s: %w", nodeName, err)
	}
	return nil
}

// DrainOptionsFromCluster converts the YAML drain config into DrainOptions.
func DrainOptionsFromCluster(enabled bool, timeout string, gracePeriod int, deleteEmptyDir, ignoreDaemonSets, disableEviction bool) DrainOptions {
	t, err := time.ParseDuration(timeout)
	if err != nil {
		t = 5 * time.Minute
	}
	return DrainOptions{
		Timeout:          t,
		GracePeriodSecs:  gracePeriod,
		DeleteEmptyDir:   deleteEmptyDir,
		IgnoreDaemonSets: ignoreDaemonSets,
		DisableEviction:  disableEviction,
		Force:            true,
	}
}
