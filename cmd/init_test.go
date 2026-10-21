package cmd

import (
	"bytes"
	"flag"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestInitTemplate_DefaultValues(t *testing.T) {
	data := initData{
		Name:    "k3s-cluster",
		Address: "10.0.0.1",
		User:    "root",
		Port:    22,
		KeyPath: "~/.ssh/id_rsa",
		Version: "v1.29.0+k3s1",
		Agent:   false,
	}

	tmpl, err := template.New("init").Parse(initTemplate)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "apiVersion: k3sctl.k3sproject.io/v1beta1")
	assert.Contains(t, output, "kind: Cluster")
	assert.Contains(t, output, "name: k3s-cluster")
	assert.Contains(t, output, "role: server")
	assert.Contains(t, output, "address: 10.0.0.1")
	assert.Contains(t, output, "user: root")
	assert.Contains(t, output, "port: 22")
	assert.Contains(t, output, `version: "v1.29.0+k3s1"`)
	assert.NotContains(t, output, "role: agent")
}

func TestInitTemplate_WithAgent(t *testing.T) {
	data := initData{
		Name:         "test-cluster",
		Address:      "192.168.1.10",
		AgentAddress: "192.168.1.11",
		User:         "ubuntu",
		Port:         2222,
		KeyPath:      "/home/user/.ssh/key",
		Version:      "v1.28.4+k3s2",
		Agent:        true,
	}

	tmpl, err := template.New("init").Parse(initTemplate)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "name: test-cluster")
	assert.Contains(t, output, "role: agent")
	assert.Contains(t, output, "address: 192.168.1.11")
	assert.Contains(t, output, "user: ubuntu")
	assert.Contains(t, output, "port: 2222")

	serverCount := strings.Count(output, "role: server")
	agentCount := strings.Count(output, "role: agent")
	assert.Equal(t, 1, serverCount)
	assert.Equal(t, 1, agentCount)
}

func TestInitTemplate_EmptyVersion(t *testing.T) {
	data := initData{
		Name:    "k3s-cluster",
		Address: "10.0.0.1",
		User:    "root",
		Port:    22,
		KeyPath: "~/.ssh/id_rsa",
		Version: "",
	}

	tmpl, err := template.New("init").Parse(initTemplate)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `version: ""`)
}

func TestInitTemplate_ValidYAML(t *testing.T) {
	data := initData{
		Name:    "k3s-cluster",
		Address: "10.0.0.1",
		User:    "root",
		Port:    22,
		KeyPath: "~/.ssh/id_rsa",
		Version: "v1.29.0+k3s1",
	}

	tmpl, err := template.New("init").Parse(initTemplate)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		assert.False(t, strings.HasPrefix(line, "\t"), "YAML should not use tabs: %q", line)
	}
}

func TestInitCommand_ReturnsCommand(t *testing.T) {
	cmd := initCommand()
	assert.Equal(t, "init", cmd.Name)
	assert.NotEmpty(t, cmd.Usage)
	assert.NotNil(t, cmd.Action)
	assert.True(t, len(cmd.Flags) > 0)
}

func TestInitCommand_HasExpectedFlags(t *testing.T) {
	cmd := initCommand()
	require.NotNil(t, cmd)

	flagNames := make(map[string]bool)
	for _, f := range cmd.Flags {
		for _, name := range f.Names() {
			flagNames[name] = true
		}
	}

	assert.True(t, flagNames["name"], "should have --name flag")
	assert.True(t, flagNames["address"], "should have --address flag")
	assert.True(t, flagNames["user"], "should have --user flag")
	assert.True(t, flagNames["port"], "should have --port flag")
	assert.True(t, flagNames["key-path"], "should have --key-path flag")
	assert.True(t, flagNames["k3s-version"], "should have --k3s-version flag")
	assert.True(t, flagNames["agent"], "should have --agent flag")
	assert.True(t, flagNames["agent-address"], "should have --agent-address flag")
}

func TestInitCommand_DefaultValues(t *testing.T) {
	cmd := initCommand()
	require.NotNil(t, cmd)

	app := cli.NewApp()
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	for _, f := range cmd.Flags {
		require.NoError(t, f.Apply(set))
	}
	ctx := cli.NewContext(app, set, nil)

	assert.Equal(t, "k3s-cluster", ctx.String("name"))
	assert.Equal(t, "10.0.0.1", ctx.String("address"))
	assert.Equal(t, "root", ctx.String("user"))
	assert.Equal(t, 22, ctx.Int("port"))
	assert.Equal(t, "~/.ssh/id_rsa", ctx.String("key-path"))
	assert.False(t, ctx.Bool("agent"))
	assert.Equal(t, "10.0.0.2", ctx.String("agent-address"))
}
