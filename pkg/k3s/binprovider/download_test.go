package binprovider

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGitHubURL(t *testing.T) {
	tests := []struct {
		name    string
		arch    string
		archErr error
		want    string
	}{
		{
			name: "amd64",
			arch: "amd64",
			want: "https://github.com/k3s-io/k3s/releases/download/v1.30.0+k3s1/k3s",
		},
		{
			name: "arm64",
			arch: "arm64",
			want: "https://github.com/k3s-io/k3s/releases/download/v1.30.0+k3s1/k3s-arm64",
		},
		{
			name: "armhf",
			arch: "arm",
			want: "https://github.com/k3s-io/k3s/releases/download/v1.30.0+k3s1/k3s-armhf",
		},
		{
			name:    "arch detection failure falls back to amd64",
			archErr: errors.New("uname failed"),
			want:    "https://github.com/k3s-io/k3s/releases/download/v1.30.0+k3s1/k3s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := &FakeHost{}
			host.ArchReturns(tt.arch, tt.archErr)

			p := NewGitHub(host, "v1.30.0+k3s1")
			assert.Equal(t, tt.want, p.url)
			assert.False(t, p.IsUpload())
		})
	}
}

func TestNewCustomURL(t *testing.T) {
	p := NewCustomURL(&FakeHost{}, "v1.30.0+k3s1", "https://mirror.example.com/k3s")

	assert.Equal(t, "https://mirror.example.com/k3s", p.url)
	assert.False(t, p.IsUpload())
}

func TestExistingProvider(t *testing.T) {
	p := NewExisting(&FakeHost{})

	needsUpgrade, err := p.NeedsUpgrade(context.Background())
	assert.NoError(t, err)
	assert.False(t, needsUpgrade)
	assert.Empty(t, p.StagedPath())
}
