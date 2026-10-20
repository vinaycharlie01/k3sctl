package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestDrainOptionToKubectlArgs_Defaults(t *testing.T) {
	opts := (&Options{}).GetDrainOptions()
	assert.Equal(t, "--force --timeout=5m --delete-emptydir-data --ignore-daemonsets", opts.ToKubectlArgs())
}

func TestDrainOptionToKubectlArgs_AllFields(t *testing.T) {
	opts := &DrainOption{
		Enabled:          true,
		Timeout:          "10m",
		GracePeriod:      30,
		DeleteEmptyDir:   false,
		IgnoreDaemonSets: false,
		DisableEviction:  true,
	}
	assert.Equal(t, "--force --grace-period=30 --timeout=10m --disable-eviction", opts.ToKubectlArgs())
}

func TestDrainOptionUnmarshal_AppliesDefaults(t *testing.T) {
	var o Options
	require.NoError(t, yaml.Unmarshal([]byte("drain:\n  timeout: 1m\n"), &o))
	require.NotNil(t, o.Drain)
	assert.True(t, o.Drain.Enabled)
	assert.True(t, o.Drain.DeleteEmptyDir)
	assert.True(t, o.Drain.IgnoreDaemonSets)
	assert.Equal(t, -1, o.Drain.GracePeriod)
	assert.Equal(t, "1m", o.Drain.Timeout)
}

func TestDrainOptionUnmarshal_OverridesDefaults(t *testing.T) {
	var o Options
	require.NoError(t, yaml.Unmarshal([]byte("drain:\n  enabled: false\n  ignoreDaemonSets: false\n"), &o))
	require.NotNil(t, o.Drain)
	assert.False(t, o.Drain.Enabled)
	assert.False(t, o.Drain.IgnoreDaemonSets)
	assert.True(t, o.Drain.DeleteEmptyDir)
}
