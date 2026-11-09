package k8s

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps a Kubernetes clientset and provides the operations k3sctl
// previously performed by shelling out to kubectl on a remote host.
type Client interface {
	DrainNode(ctx context.Context, nodeName string, opts DrainOptions) error
	CordonNode(ctx context.Context, nodeName string) error
	UncordonNode(ctx context.Context, nodeName string) error
	DeleteNode(ctx context.Context, nodeName string) error
	ApplyManifest(ctx context.Context, manifest []byte) error
	GetNodeStatus(ctx context.Context, nodeName string) (*NodeStatus, error)
	IsNodeReady(ctx context.Context, nodeName string) (bool, error)
	CountReadyNodes(ctx context.Context) (int, error)
	CheckHealth(ctx context.Context) error
	Clientset() kubernetes.Interface
}

// NodeStatus represents the readiness state of a Kubernetes node.
type NodeStatus struct {
	Name  string
	Ready bool
}

type client struct {
	cs      kubernetes.Interface
	restCfg *rest.Config
}

// NewFromKubeconfig builds a Client from raw kubeconfig YAML bytes, the same
// format that phase/get_kubeconfig.go fetches from the remote server.
func NewFromKubeconfig(kubeconfig []byte) (Client, error) {
	restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("parse kubeconfig: %w", err)
	}
	restCfg.Timeout = 30 * time.Second

	cs, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes client: %w", err)
	}
	return &client{cs: cs, restCfg: restCfg}, nil
}

// NewFromClientset wraps an existing clientset — used for testing with fakes.
func NewFromClientset(cs kubernetes.Interface) Client {
	return &client{cs: cs}
}

func (c *client) Clientset() kubernetes.Interface {
	return c.cs
}

func (c *client) GetNodeStatus(ctx context.Context, nodeName string) (*NodeStatus, error) {
	node, err := c.cs.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get node %s: %w", nodeName, err)
	}
	return &NodeStatus{
		Name:  node.Name,
		Ready: isNodeReady(node),
	}, nil
}

func (c *client) IsNodeReady(ctx context.Context, nodeName string) (bool, error) {
	status, err := c.GetNodeStatus(ctx, nodeName)
	if err != nil {
		return false, err
	}
	return status.Ready, nil
}

func (c *client) CountReadyNodes(ctx context.Context) (int, error) {
	nodes, err := c.cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, fmt.Errorf("list nodes: %w", err)
	}
	count := 0
	for i := range nodes.Items {
		if isNodeReady(&nodes.Items[i]) {
			count++
		}
	}
	return count, nil
}

func (c *client) DeleteNode(ctx context.Context, nodeName string) error {
	err := c.cs.CoreV1().Nodes().Delete(ctx, nodeName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("delete node %s: %w", nodeName, err)
	}
	return nil
}

func (c *client) CheckHealth(ctx context.Context) error {
	if c.restCfg == nil {
		_, err := c.cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
		return err
	}
	restClient, err := rest.RESTClientFor(healthCheckConfig(c.restCfg))
	if err != nil {
		return fmt.Errorf("create REST client: %w", err)
	}
	result := restClient.Get().AbsPath("/healthz").Do(ctx)
	if result.Error() != nil {
		return fmt.Errorf("healthz: %w", result.Error())
	}
	body, err := result.Raw()
	if err != nil {
		return fmt.Errorf("read healthz response: %w", err)
	}
	if string(body) != "ok" {
		return fmt.Errorf("healthz returned: %s", string(body))
	}
	return nil
}

func healthCheckConfig(base *rest.Config) *rest.Config {
	cfg := rest.CopyConfig(base)
	cfg.APIPath = "/"
	cfg.GroupVersion = &corev1.SchemeGroupVersion
	cfg.NegotiatedSerializer = nil
	return cfg
}

func isNodeReady(node *corev1.Node) bool {
	for _, c := range node.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}
