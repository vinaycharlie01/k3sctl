package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalFlags(t *testing.T) {
	flags := GlobalFlags()
	require.NotEmpty(t, flags)

	names := make(map[string]bool)
	for _, f := range flags {
		for _, n := range f.Names() {
			names[n] = true
		}
	}

	assert.True(t, names["debug"], "should have debug flag")
	assert.True(t, names["trace"], "should have trace flag")
	assert.True(t, names["no-redact"], "should have no-redact flag")
}

func TestConfigFlag(t *testing.T) {
	f := configFlag()
	require.NotNil(t, f)
	assert.Equal(t, "config", f.Name)
	assert.Equal(t, "k3sctl.yaml", f.Value)
	assert.Contains(t, f.Aliases, "c")
	assert.Contains(t, f.EnvVars, "K3SCTL_CONFIG")
}

func TestConcurrencyFlag(t *testing.T) {
	f := concurrencyFlag()
	require.NotNil(t, f)
	assert.Equal(t, "concurrency", f.Name)
	assert.Equal(t, 30, f.Value)
}

func TestConcurrentUploadsFlag(t *testing.T) {
	f := concurrentUploadsFlag()
	require.NotNil(t, f)
	assert.Equal(t, "concurrent-uploads", f.Name)
	assert.Equal(t, 5, f.Value)
}

func TestDryRunFlag(t *testing.T) {
	f := dryRunFlag()
	require.NotNil(t, f)
	assert.Equal(t, "dry-run", f.Name)
}

func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
