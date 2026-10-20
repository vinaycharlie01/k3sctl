package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlagsAdd(t *testing.T) {
	var f Flags
	f.Add("--tls-san=10.0.0.1")
	f.Add("--disable=traefik")
	assert.Len(t, f, 2)
	assert.Equal(t, "--tls-san=10.0.0.1", f[0])
}

func TestFlagsAddOrReplace(t *testing.T) {
	f := Flags{"--tls-san=10.0.0.1"}
	f.AddOrReplace("--tls-san", "10.0.0.2")
	assert.Len(t, f, 1)
	assert.Equal(t, "--tls-san=10.0.0.2", f[0])
}

func TestFlagsDelete(t *testing.T) {
	f := Flags{"--tls-san=10.0.0.1", "--disable=traefik"}
	f.Delete("--tls-san")
	assert.Len(t, f, 1)
	assert.Equal(t, "--disable=traefik", f[0])
}

func TestFlagsGetValue(t *testing.T) {
	f := Flags{"--tls-san=10.0.0.1", "--cluster-init"}

	val, ok := f.GetValue("--tls-san")
	require.True(t, ok)
	assert.Equal(t, "10.0.0.1", val)

	val, ok = f.GetValue("--cluster-init")
	require.True(t, ok)
	assert.Equal(t, "", val)

	_, ok = f.GetValue("--nonexistent")
	assert.False(t, ok)
}

func TestFlagsEquals(t *testing.T) {
	a := Flags{"--b=2", "--a=1"}
	b := Flags{"--a=1", "--b=2"}
	assert.True(t, a.Equals(b))

	c := Flags{"--a=1"}
	assert.False(t, a.Equals(c))
}

func TestFlagsMap(t *testing.T) {
	f := Flags{"--tls-san=10.0.0.1", "--cluster-init"}
	m := f.Map()
	assert.Equal(t, "10.0.0.1", m["--tls-san"])
	assert.Equal(t, "", m["--cluster-init"])
}

func TestFlagsFromMap(t *testing.T) {
	m := map[string]string{"--tls-san": "10.0.0.1", "--cluster-init": ""}
	f := FlagsFromMap(m)
	assert.Len(t, f, 2)
	assert.True(t, f.Include("--tls-san"))
	assert.True(t, f.Include("--cluster-init"))
}
