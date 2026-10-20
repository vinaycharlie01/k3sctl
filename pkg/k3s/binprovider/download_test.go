package binprovider_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vinaycharlie01/k3sctl/pkg/k3s/binprovider"
	"github.com/vinaycharlie01/k3sctl/pkg/k3s/binprovider/mocks"
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
			host := &mocks.FakeHost{}
			host.ArchReturns(tt.arch, tt.archErr)

			p := binprovider.NewGitHub(host, "v1.30.0+k3s1")
			assert.False(t, p.IsUpload())

			assert.NoError(t, p.Stage(context.Background()))

			url, _ := host.DownloadURLArgsForCall(0)
			assert.Equal(t, tt.want, url)
		})
	}
}

func TestNewCustomURL(t *testing.T) {
	host := &mocks.FakeHost{}
	p := binprovider.NewCustomURL(host, "v1.30.0+k3s1", "https://mirror.example.com/k3s")

	assert.False(t, p.IsUpload())
	assert.NoError(t, p.Stage(context.Background()))

	url, _ := host.DownloadURLArgsForCall(0)
	assert.Equal(t, "https://mirror.example.com/k3s", url)
}

func TestExistingProvider(t *testing.T) {
	p := binprovider.NewExisting(&mocks.FakeHost{})

	needsUpgrade, err := p.NeedsUpgrade(context.Background())
	assert.NoError(t, err)
	assert.False(t, needsUpgrade)
	assert.Empty(t, p.StagedPath())
}

func TestNeedsUpgrade(t *testing.T) {
	tests := []struct {
		name      string
		installed string
		target    string
		want      bool
	}{
		{"fresh host", "", "v1.30.0+k3s1", true},
		{"same version", "v1.30.0+k3s1", "v1.30.0+k3s1", false},
		{"different version", "v1.29.0+k3s1", "v1.30.0+k3s1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := &mocks.FakeHost{}
			host.InstalledK3sVersionReturns(tt.installed)

			got, err := binprovider.NewGitHub(host, tt.target).NeedsUpgrade(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
