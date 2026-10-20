package phase

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemdEnvironment(t *testing.T) {
	assert.Empty(t, systemdEnvironment(nil))

	got := systemdEnvironment(map[string]string{"B": "2", "A": "1"})
	assert.Equal(t, "Environment=\"A=1\"\nEnvironment=\"B=2\"\n", got)
}

func TestK3sServerUnitRendersEnvironment(t *testing.T) {
	unit := k3sServerUnit("/usr/local/bin/k3s", []string{"--cluster-init"}, map[string]string{
		"ETCD_UNSUPPORTED_ARCH": "arm",
	})

	assert.True(t, strings.Contains(unit, "Environment=\"ETCD_UNSUPPORTED_ARCH=arm\"\n"))
	assert.True(t, strings.HasPrefix(unit, "[Unit]\n"))
	assert.True(t, strings.Contains(unit, "ExecStart=/usr/local/bin/k3s server --cluster-init\n"))
}

func TestK3sAgentUnitRendersEnvironment(t *testing.T) {
	unit := k3sAgentUnit("/usr/local/bin/k3s", nil, map[string]string{"FOO": "bar"})

	assert.True(t, strings.Contains(unit, "Environment=\"FOO=bar\"\n"))
	assert.True(t, strings.Contains(unit, "ExecStart=/usr/local/bin/k3s agent\n"))
}
