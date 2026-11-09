package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestIsNodeReady(t *testing.T) {
	cs := fake.NewSimpleClientset(fakeNode("node1", true), fakeNode("node2", false))
	c := NewFromClientset(cs)
	ctx := context.Background()

	ready, err := c.IsNodeReady(ctx, "node1")
	require.NoError(t, err)
	assert.True(t, ready)

	ready, err = c.IsNodeReady(ctx, "node2")
	require.NoError(t, err)
	assert.False(t, ready)

	_, err = c.IsNodeReady(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestCountReadyNodes(t *testing.T) {
	cs := fake.NewSimpleClientset(
		fakeNode("s1", true),
		fakeNode("s2", true),
		fakeNode("a1", false),
	)
	c := NewFromClientset(cs)
	ctx := context.Background()

	count, err := c.CountReadyNodes(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestDeleteNode(t *testing.T) {
	cs := fake.NewSimpleClientset(fakeNode("node1", true))
	c := NewFromClientset(cs)
	ctx := context.Background()

	err := c.DeleteNode(ctx, "node1")
	require.NoError(t, err)

	_, err = cs.CoreV1().Nodes().Get(ctx, "node1", metav1.GetOptions{})
	assert.Error(t, err)
}

func TestDeleteNode_NotFound(t *testing.T) {
	cs := fake.NewSimpleClientset()
	c := NewFromClientset(cs)
	err := c.DeleteNode(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestGetNodeStatus(t *testing.T) {
	cs := fake.NewSimpleClientset(fakeNode("server1", true))
	c := NewFromClientset(cs)

	status, err := c.GetNodeStatus(context.Background(), "server1")
	require.NoError(t, err)
	assert.Equal(t, "server1", status.Name)
	assert.True(t, status.Ready)
}

func TestCordonUncordon(t *testing.T) {
	cs := fake.NewSimpleClientset(fakeNode("node1", true))
	c := NewFromClientset(cs)
	ctx := context.Background()

	err := c.CordonNode(ctx, "node1")
	require.NoError(t, err)

	node, err := cs.CoreV1().Nodes().Get(ctx, "node1", metav1.GetOptions{})
	require.NoError(t, err)
	assert.True(t, node.Spec.Unschedulable)

	err = c.UncordonNode(ctx, "node1")
	require.NoError(t, err)

	node, err = cs.CoreV1().Nodes().Get(ctx, "node1", metav1.GetOptions{})
	require.NoError(t, err)
	assert.False(t, node.Spec.Unschedulable)
}

func TestCheckHealth_FakeClient(t *testing.T) {
	cs := fake.NewSimpleClientset(fakeNode("n1", true))
	c := NewFromClientset(cs)
	err := c.CheckHealth(context.Background())
	require.NoError(t, err)
}
