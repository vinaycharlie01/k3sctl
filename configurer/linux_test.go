package configurer_test

import (
	"testing"

	rigos "github.com/k0sproject/rig/v2/os"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vinaycharlie01/k3sctl/configurer"

	_ "github.com/vinaycharlie01/k3sctl/configurer/linux"
	_ "github.com/vinaycharlie01/k3sctl/configurer/linux/enterpriselinux"
)

func TestResolveKnownDistros(t *testing.T) {
	distros := []struct {
		id   string
		name string
	}{
		{"ubuntu", "Ubuntu"},
		{"debian", "Debian"},
		{"centos", "CentOS"},
		{"rocky", "Rocky Linux"},
		{"almalinux", "AlmaLinux"},
		{"amzn", "Amazon Linux"},
		{"ol", "Oracle Linux"},
		{"alpine", "Alpine Linux"},
		{"flatcar", "Flatcar Container Linux"},
		{"sles", "SUSE Linux Enterprise Server"},
		{"opensuse", "openSUSE"},
	}

	for _, d := range distros {
		t.Run(d.id, func(t *testing.T) {
			release := &rigos.Release{ID: d.id, Name: d.name}
			c, err := configurer.ResolveOSModule(release)
			require.NoError(t, err, "should resolve OS module for %s", d.id)
			assert.NotNil(t, c)
			assert.Equal(t, "linux", c.Kind())
		})
	}
}

func TestResolveFedora(t *testing.T) {
	release := &rigos.Release{ID: "fedora", Name: "Fedora Linux"}
	c, err := configurer.ResolveOSModule(release)
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestResolveRHEL(t *testing.T) {
	release := &rigos.Release{ID: "rhel", Name: "Red Hat Enterprise Linux"}
	c, err := configurer.ResolveOSModule(release)
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestResolveUnknownOS(t *testing.T) {
	release := &rigos.Release{ID: "unknowndistro", Name: "Unknown"}
	_, err := configurer.ResolveOSModule(release)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported OS")
}

func TestResolveViaIDLike(t *testing.T) {
	release := &rigos.Release{
		ID:     "pop",
		Name:   "Pop!_OS",
		IDLike: []string{"ubuntu", "debian"},
	}
	c, err := configurer.ResolveOSModule(release)
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestResolveOpenSUSEMicroOS(t *testing.T) {
	release := &rigos.Release{ID: "opensuse-microos", Name: "openSUSE MicroOS"}
	c, err := configurer.ResolveOSModule(release)
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestLinuxPaths(t *testing.T) {
	l := &configurer.Linux{}
	assert.Equal(t, "/usr/local/bin/k3s", l.K3sBinaryPath())
	assert.Equal(t, "/etc/rancher/k3s/config.yaml", l.K3sConfigPath())
	assert.Equal(t, "/var/lib/rancher/k3s/server/node-token", l.K3sTokenPath())
	assert.Equal(t, "/var/lib/rancher/k3s", l.DataDirDefaultPath())
	assert.Equal(t, "/var/lib/rancher/k3s", l.K3sDataDirDefaultPath())
	assert.Equal(t, "/etc/rancher/k3s", l.K3sConfigDirPath())
	assert.Equal(t, "/var/lock/k3sctl.lock", l.K3sctlLockFilePath())
}

func TestLinuxSetPath(t *testing.T) {
	l := &configurer.Linux{}
	assert.Equal(t, "/usr/local/bin/k3s", l.K3sBinaryPath())
	l.SetPath("K3sBinaryPath", "/opt/k3s/bin/k3s")
	assert.Equal(t, "/opt/k3s/bin/k3s", l.K3sBinaryPath())
}

func TestLinuxKind(t *testing.T) {
	l := &configurer.Linux{}
	assert.Equal(t, "linux", l.Kind())
	assert.Equal(t, "linux", l.OSKind())
}

func TestLinuxK3sCmdf(t *testing.T) {
	l := &configurer.Linux{}
	cmd := l.K3sCmdf("server --cluster-init")
	assert.Equal(t, "/usr/local/bin/k3s server --cluster-init", cmd)
}

func TestConfigurerInterface(t *testing.T) {
	release := &rigos.Release{ID: "ubuntu", Name: "Ubuntu"}
	c, err := configurer.ResolveOSModule(release)
	require.NoError(t, err)

	assert.Equal(t, "linux", c.Kind())
	assert.Equal(t, "linux", c.OSKind())
	assert.Equal(t, "/usr/local/bin/k3s", c.K3sBinaryPath())
	assert.Equal(t, "/etc/rancher/k3s/config.yaml", c.K3sConfigPath())
	assert.Equal(t, "/etc/rancher/k3s", c.K3sConfigDirPath())
	assert.Equal(t, "/var/lib/rancher/k3s", c.K3sDataDirDefaultPath())
	assert.Equal(t, "/var/lib/rancher/k3s/server/node-token", c.K3sTokenPath())
	assert.Equal(t, "/var/lock/k3sctl.lock", c.K3sctlLockFilePath())
}
