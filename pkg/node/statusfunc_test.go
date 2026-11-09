package node

import (
	"context"
	"testing"

	"github.com/nirantarasi/k3sctl/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func fakeNode(name string, ready bool) *corev1.Node {
	status := corev1.ConditionFalse
	if ready {
		status = corev1.ConditionTrue
	}
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: status},
			},
		},
	}
}

func TestKubeNodeReadyFunc_Ready(t *testing.T) {
	cs := fake.NewSimpleClientset(fakeNode("node1", true))
	kc := k8s.NewFromClientset(cs)

	fn := KubeNodeReadyFunc(kc, "node1")
	err := fn(context.Background())
	if err != nil {
		t.Errorf("expected no error for ready node, got: %v", err)
	}
}

func TestKubeNodeReadyFunc_NotReady(t *testing.T) {
	cs := fake.NewSimpleClientset(fakeNode("node1", false))
	kc := k8s.NewFromClientset(cs)

	fn := KubeNodeReadyFunc(kc, "node1")
	err := fn(context.Background())
	if err == nil {
		t.Error("expected error for not-ready node")
	}
}

func TestKubeNodeReadyFunc_NotFound(t *testing.T) {
	cs := fake.NewSimpleClientset()
	kc := k8s.NewFromClientset(cs)

	fn := KubeNodeReadyFunc(kc, "missing")
	err := fn(context.Background())
	if err == nil {
		t.Error("expected error for missing node")
	}
}

func TestKubeNodeReadyFunc_NilClient(t *testing.T) {
	fn := KubeNodeReadyFunc(nil, "node1")
	err := fn(context.Background())
	if err == nil {
		t.Error("expected error for nil client")
	}
}

func TestKubeNodeReadyFunc_MultipleConditions(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node1"},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeDiskPressure, Status: corev1.ConditionFalse},
				{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionFalse},
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	}
	cs := fake.NewSimpleClientset(node)
	kc := k8s.NewFromClientset(cs)

	fn := KubeNodeReadyFunc(kc, "node1")
	if err := fn(context.Background()); err != nil {
		t.Errorf("expected no error for ready node with multiple conditions, got: %v", err)
	}
}
