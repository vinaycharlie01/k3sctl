package cmd

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestApplyCommand_HasExpectedFlags(t *testing.T) {
	cmd := applyCommand()
	require.NotNil(t, cmd)
	assert.Equal(t, "apply", cmd.Name)

	flagNames := make(map[string]bool)
	for _, f := range cmd.Flags {
		for _, name := range f.Names() {
			flagNames[name] = true
		}
	}

	assert.True(t, flagNames["config"], "should have --config flag")
	assert.True(t, flagNames["no-drain"], "should have --no-drain flag")
	assert.True(t, flagNames["no-wait"], "should have --no-wait flag")
	assert.True(t, flagNames["dry-run"], "should have --dry-run flag")
	assert.True(t, flagNames["restore-from"], "should have --restore-from flag")
	assert.True(t, flagNames["kubeconfig-out"], "should have --kubeconfig-out flag")
	assert.True(t, flagNames["kubeconfig-api-address"], "should have --kubeconfig-api-address flag")
	assert.True(t, flagNames["disable-downgrade-check"], "should have --disable-downgrade-check flag")
	assert.True(t, flagNames["force"], "should have --force flag")
	assert.True(t, flagNames["concurrency"], "should have --concurrency flag")
	assert.True(t, flagNames["concurrent-uploads"], "should have --concurrent-uploads flag")
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := loadConfig("/nonexistent/path/k3sctl.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := tmp + "/bad.yaml"
	require.NoError(t, writeTestFile(cfgPath, "not: valid: yaml: {{{}}}"))
	_, err := loadConfig(cfgPath)
	require.Error(t, err)
}

func TestLoadConfig_ValidMinimalConfig(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := tmp + "/k3sctl.yaml"
	content := `apiVersion: k3sctl.k3sproject.io/v1beta1
kind: Cluster
metadata:
  name: test
spec:
  hosts:
    - role: server
      localhost: true
  k3s:
    version: "v1.28.0+k3s1"
`
	require.NoError(t, writeTestFile(cfgPath, content))
	cfg, err := loadConfig(cfgPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "test", cfg.Metadata.Name)
	assert.Len(t, cfg.Spec.Hosts, 1)
}

func TestDisplayLogo(t *testing.T) {
	// Just ensure it doesn't panic
	displayLogo()
}

func TestInitLogging_DebugLevel(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	set.Bool("debug", true, "")
	set.Bool("trace", false, "")
	ctx := cli.NewContext(app, set, nil)

	require.NoError(t, initLogging(ctx))
}

func TestInitLogging_TraceLevel(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	set.Bool("debug", false, "")
	set.Bool("trace", true, "")
	ctx := cli.NewContext(app, set, nil)

	require.NoError(t, initLogging(ctx))
}
