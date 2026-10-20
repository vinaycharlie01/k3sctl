package manifest

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validDoc = `apiVersion: k3sctl.k3sproject.io/v1beta1
kind: Cluster
metadata:
  name: test-cluster
spec:
  hosts:
    - role: server
`

const multiDoc = `apiVersion: k3sctl.k3sproject.io/v1beta1
kind: Cluster
metadata:
  name: cluster-one
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
`

func TestParseValidDocument(t *testing.T) {
	r := &Reader{}
	err := r.Parse(strings.NewReader(validDoc))
	require.NoError(t, err)
	require.Equal(t, 1, r.Len())

	res := r.Resources()
	assert.Equal(t, "k3sctl.k3sproject.io/v1beta1", res[0].APIVersion)
	assert.Equal(t, "Cluster", res[0].Kind)
	assert.Equal(t, "test-cluster", res[0].Metadata.Name)
}

func TestParseMultipleDocuments(t *testing.T) {
	r := &Reader{}
	err := r.Parse(strings.NewReader(multiDoc))
	require.NoError(t, err)
	require.Equal(t, 2, r.Len())

	res := r.Resources()
	assert.Equal(t, "Cluster", res[0].Kind)
	assert.Equal(t, "ConfigMap", res[1].Kind)
}

func TestParseEmptyInput(t *testing.T) {
	r := &Reader{}
	err := r.Parse(strings.NewReader(""))
	require.NoError(t, err)
	assert.Equal(t, 0, r.Len())
}

func TestParseWhitespaceOnly(t *testing.T) {
	r := &Reader{}
	err := r.Parse(strings.NewReader("   \n\n  \n"))
	require.NoError(t, err)
	assert.Equal(t, 0, r.Len())
}

func TestParseInvalidYAML(t *testing.T) {
	r := &Reader{}
	err := r.Parse(strings.NewReader("{{invalid yaml}}"))
	assert.Error(t, err)
}

func TestParseMissingAPIVersion(t *testing.T) {
	doc := `kind: Cluster
metadata:
  name: test
`
	r := &Reader{}
	err := r.Parse(strings.NewReader(doc))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing apiVersion or kind")
}

func TestParseMissingKind(t *testing.T) {
	doc := `apiVersion: v1
metadata:
  name: test
`
	r := &Reader{}
	err := r.Parse(strings.NewReader(doc))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing apiVersion or kind")
}

func TestParseIgnoreErrors(t *testing.T) {
	doc := `kind: NoApiVersion
metadata:
  name: test
`
	r := &Reader{IgnoreErrors: true}
	err := r.Parse(strings.NewReader(doc))
	assert.NoError(t, err)
	assert.Equal(t, 0, r.Len())
}

func TestParseString(t *testing.T) {
	r := &Reader{}
	err := r.ParseString(validDoc)
	require.NoError(t, err)
	assert.Equal(t, 1, r.Len())
}

func TestParseBytes(t *testing.T) {
	r := &Reader{}
	err := r.ParseBytes([]byte(validDoc))
	require.NoError(t, err)
	assert.Equal(t, 1, r.Len())
}

func TestParseBytesWithOrigin(t *testing.T) {
	r := &Reader{}
	err := r.ParseBytesWithOrigin([]byte(validDoc), "my-config.yaml")
	require.NoError(t, err)
	require.Equal(t, 1, r.Len())
	assert.Equal(t, "my-config.yaml", r.Resources()[0].Origin)
}

func TestWithOriginOption(t *testing.T) {
	r := &Reader{}
	err := r.Parse(strings.NewReader(validDoc), WithOrigin("custom-origin.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "custom-origin.yaml", r.Resources()[0].Origin)
}

func TestResourceDefinitionFilename(t *testing.T) {
	rd := &ResourceDefinition{
		Kind:   "Cluster",
		Origin: "/path/to/my-cluster.yaml",
	}
	rd.Metadata.Name = "test"
	assert.Equal(t, "my-cluster.yaml", rd.Filename())

	rd2 := &ResourceDefinition{
		Kind:   "Cluster",
		Origin: "stdin",
	}
	rd2.Metadata.Name = "test-cluster"
	assert.Equal(t, "Cluster-test-cluster.yaml", rd2.Filename())
}

func TestResourceDefinitionUnmarshal(t *testing.T) {
	r := &Reader{}
	require.NoError(t, r.ParseString(validDoc))
	res := r.Resources()[0]

	var out struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
		Metadata   struct {
			Name string `yaml:"name"`
		} `yaml:"metadata"`
		Spec map[string]any `yaml:"spec"`
	}
	require.NoError(t, res.Unmarshal(&out))
	assert.Equal(t, "k3sctl.k3sproject.io/v1beta1", out.APIVersion)
	assert.Equal(t, "Cluster", out.Kind)
	assert.Equal(t, "test-cluster", out.Metadata.Name)
}

func TestResourceDefinitionReaderAndBytes(t *testing.T) {
	r := &Reader{}
	require.NoError(t, r.ParseString(validDoc))
	res := r.Resources()[0]

	assert.NotEmpty(t, res.Bytes())
	reader := res.Reader()
	assert.NotNil(t, reader)
	assert.Greater(t, reader.Len(), 0)
}

func TestGetResources(t *testing.T) {
	r := &Reader{}
	require.NoError(t, r.ParseString(multiDoc))

	found, err := r.GetResources("k3sctl.k3sproject.io/v1beta1", "Cluster")
	require.NoError(t, err)
	assert.Len(t, found, 1)

	_, err = r.GetResources("nonexistent", "Nothing")
	assert.Error(t, err)
}

func TestFilterResources(t *testing.T) {
	r := &Reader{}
	require.NoError(t, r.ParseString(multiDoc))

	filtered := r.FilterResources(func(rd *ResourceDefinition) bool {
		return rd.Kind == "ConfigMap"
	})
	assert.Len(t, filtered, 1)
	assert.Equal(t, "my-config", filtered[0].Metadata.Name)
}

func TestMultipleParseCalls(t *testing.T) {
	r := &Reader{}
	require.NoError(t, r.ParseString(validDoc))
	require.NoError(t, r.ParseString(validDoc))
	assert.Equal(t, 2, r.Len())
}
