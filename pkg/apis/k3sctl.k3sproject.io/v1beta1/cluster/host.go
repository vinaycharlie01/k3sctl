package cluster

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	rig "github.com/k0sproject/rig/v2"
	"github.com/k0sproject/rig/v2/remotefs"
	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/configurer"
)

type BinaryProvider interface {
	NeedsUpgrade(ctx context.Context) (bool, error)
	IsUpload() bool
	Stage(ctx context.Context) error
	CleanUp(ctx context.Context) error
}

type Role string

const (
	RoleServer Role = "server"
	RoleAgent  Role = "agent"
)

type Host struct {
	rig.ClientWithConfig `yaml:",inline"`

	Role             Role              `yaml:"role" default:"server"`
	PrivateAddress   string            `yaml:"privateAddress,omitempty"`
	PrivateInterface string            `yaml:"privateInterface,omitempty"`
	Hostname         string            `yaml:"hostname,omitempty"`
	NoFlannel        bool              `yaml:"noFlannel,omitempty"`
	InstallFlags     Flags             `yaml:"installFlags,omitempty"`
	Environment      map[string]string `yaml:"environment,omitempty"`
	UploadBinary     bool              `yaml:"uploadBinary,omitempty"`
	K3sBinaryPath    string            `yaml:"k3sBinaryPath,omitempty"`
	DataDir          string            `yaml:"dataDir,omitempty"`
	Labels           map[string]string `yaml:"labels,omitempty"`
	Taints           []string          `yaml:"taints,omitempty"`
	Hooks            Hooks             `yaml:"hooks,omitempty"`
	UploadFiles      []UploadFile      `yaml:"files,omitempty"`
	OSIDOverride     string            `yaml:"os,omitempty"`
	Reset            bool              `yaml:"-"`

	Configurer configurer.Configurer `yaml:"-"`
	Metadata   HostMetadata          `yaml:"-"`

	configLock sync.Mutex
}

type HostMetadata struct {
	K3sRunningVersion   string
	K3sInstalledVersion string
	Arch                string
	Hostname            string
	MachineID           string
	K3sServiceRunning   bool
	K3sServiceInstalled bool
	K3sExistingConfig   string
	NodeToken           string
	Ready               bool
	IsLeader            bool
	KubernetesNodeName  string
	NeedsUpgrade        bool
	NeedsReinstall      bool
	BinaryProvider      BinaryProvider
}

func (h *Host) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type host Host
	yh := (*host)(h)
	yh.Environment = make(map[string]string)
	if err := unmarshal(yh); err != nil {
		return err
	}
	if h.Client != nil {
		h.Disconnect()
		h.Client = nil
	}
	return nil
}

func (h *Host) String() string {
	if h.Hostname != "" {
		return h.Hostname
	}
	if addr := h.Address(); addr != "" {
		return addr
	}
	return "unknown-host"
}

func (h *Host) Address() string {
	if h.ConnectionConfig.SSH != nil {
		return h.ConnectionConfig.SSH.Address
	}
	if h.ConnectionConfig.OpenSSH != nil {
		return h.ConnectionConfig.OpenSSH.Address
	}
	if h.ConnectionConfig.Localhost {
		return "localhost"
	}
	return ""
}

func (h *Host) Protocol() string {
	if h.ConnectionConfig.SSH != nil {
		return "ssh"
	}
	if h.ConnectionConfig.OpenSSH != nil {
		return "openssh"
	}
	if h.ConnectionConfig.Localhost {
		return "localhost"
	}
	return "unknown"
}

func (h *Host) IsServer() bool {
	return h.Role == RoleServer
}

func (h *Host) IsAgent() bool {
	return h.Role == RoleAgent
}

func (h *Host) Connect(ctx context.Context) error {
	if err := h.ClientWithConfig.Connect(ctx); err != nil {
		return fmt.Errorf("connect to %s: %w", h, err)
	}
	return nil
}

func (h *Host) Disconnect() {
	if h.Client != nil {
		h.Client.Disconnect()
		h.Client = nil
	}
}

func (h *Host) Sudo() *rig.Client {
	if h.Client == nil {
		return nil
	}
	return h.Client.Sudo()
}

func (h *Host) FS() remotefs.FS {
	if h.Client == nil {
		return nil
	}
	return h.Client.FS()
}

func (h *Host) K3sDataDir() string {
	if h.DataDir != "" {
		return h.DataDir
	}
	if h.Configurer != nil {
		return h.Configurer.K3sDataDirDefaultPath()
	}
	return "/var/lib/rancher/k3s"
}

func (h *Host) K3sBinaryDir() string {
	if h.K3sBinaryPath != "" {
		return h.K3sBinaryPath
	}
	if h.Configurer != nil {
		return h.Configurer.K3sBinaryPath()
	}
	return "/usr/local/bin/k3s"
}

func (h *Host) K3sConfigDir() string {
	if h.Configurer != nil {
		return h.Configurer.K3sConfigPath()
	}
	return "/etc/rancher/k3s/config.yaml"
}

func (h *Host) KubernetesNodeName() string {
	if h.Metadata.KubernetesNodeName != "" {
		return h.Metadata.KubernetesNodeName
	}
	if h.Metadata.Hostname != "" {
		return strings.ToLower(h.Metadata.Hostname)
	}
	return strings.ToLower(h.String())
}

func (h *Host) K3sServiceName() string {
	if h.IsServer() {
		return "k3s"
	}
	return "k3s-agent"
}

func (h *Host) K3sInstallFlags() Flags {
	flags := make(Flags, 0)
	flags = append(flags, h.InstallFlags...)

	for k, v := range h.Labels {
		flags.AddOrReplace("--node-label", fmt.Sprintf("%s=%s", k, v))
	}
	for _, taint := range h.Taints {
		flags.Add("--node-taint=" + taint)
	}
	return flags
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func (h *Host) DrainNode(leader *Host, opts interface{}) error {
	log.Infof("%s: draining node", h)
	cmd := leader.Configurer.KubectlCmdf(leader, leader.K3sDataDir(),
		"drain %s --ignore-daemonsets --delete-emptydir-data --timeout=5m --force",
		shellQuote(h.KubernetesNodeName()))
	return leader.Sudo().Exec(cmd)
}

func (h *Host) UncordonNode(leader *Host) error {
	log.Infof("%s: uncordoning node", h)
	cmd := leader.Configurer.KubectlCmdf(leader, leader.K3sDataDir(),
		"uncordon %s", shellQuote(h.KubernetesNodeName()))
	return leader.Sudo().Exec(cmd)
}

func (h *Host) CordonNode(leader *Host) error {
	log.Infof("%s: cordoning node", h)
	cmd := leader.Configurer.KubectlCmdf(leader, leader.K3sDataDir(),
		"cordon %s", shellQuote(h.KubernetesNodeName()))
	return leader.Sudo().Exec(cmd)
}

func (h *Host) DeleteNode(leader *Host) error {
	log.Infof("%s: deleting node from cluster", h)
	cmd := leader.Configurer.KubectlCmdf(leader, leader.K3sDataDir(),
		"delete node %s --ignore-not-found=true", shellQuote(h.KubernetesNodeName()))
	return leader.Sudo().Exec(cmd)
}

func (h *Host) CheckHTTPStatus(ctx context.Context, url string, expected ...int) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	for _, exp := range expected {
		if resp.StatusCode == exp {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
}

func (h *Host) RunHooks(action, stage string) error {
	if h.Hooks == nil {
		return nil
	}
	stageHooks, ok := h.Hooks[action]
	if !ok {
		return nil
	}
	commands, ok := stageHooks[stage]
	if !ok {
		return nil
	}

	for _, cmd := range commands {
		log.Infof("%s: running %s %s hook: %s", h, action, stage, cmd)
		if err := h.Sudo().Exec(cmd); err != nil {
			return fmt.Errorf("hook %q failed on %s: %w", cmd, h, err)
		}
	}
	return nil
}

func (h *Host) Validate() error {
	if h.Role != RoleServer && h.Role != RoleAgent {
		return fmt.Errorf("invalid role %q for host %s: must be 'server' or 'agent'", h.Role, h)
	}
	return nil
}

func (h *Host) ExpandTokens(input string) string {
	r := strings.NewReplacer(
		"{HOSTNAME}", h.Metadata.Hostname,
		"{ADDRESS}", h.Address(),
		"{ROLE}", string(h.Role),
		"{ARCH}", h.Metadata.Arch,
		"{PRIVATE_ADDRESS}", h.PrivateAddress,
	)
	return r.Replace(input)
}

func (h *Host) SetFileMode(path string, mode fs.FileMode) error {
	return h.Sudo().FS().Chmod(path, mode)
}

func (h *Host) Touch(path string, modTime time.Time) error {
	ts := modTime.Unix()
	return h.Sudo().FS().Chtimes(path, ts, ts)
}

func (h *Host) DeleteFile(path string) error {
	return h.Sudo().FS().Remove(path)
}

func (h *Host) Arch() (string, error) {
	output, err := h.ExecOutput("uname -m")
	if err != nil {
		return "", fmt.Errorf("detect architecture: %w", err)
	}
	arch := strings.TrimSpace(output)
	switch arch {
	case "x86_64":
		return "amd64", nil
	case "aarch64":
		return "arm64", nil
	case "armv7l":
		return "arm", nil
	default:
		return arch, nil
	}
}

func (h *Host) Dir(path string) (string, error) {
	output, err := h.Sudo().ExecOutput(fmt.Sprintf("dirname %s", shellQuote(path)))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (h *Host) InstalledK3sVersion() string {
	return h.Metadata.K3sInstalledVersion
}

func (h *Host) RunningK3sVersion() string {
	return h.Metadata.K3sRunningVersion
}

func (h *Host) OSKind() (string, error) {
	if h.Configurer != nil {
		return h.Configurer.OSKind(), nil
	}
	return "", fmt.Errorf("configurer not resolved for %s", h)
}

func (h *Host) DownloadURL(url, dest string) error {
	return h.Sudo().Exec(fmt.Sprintf("curl -sSfL -o %s %s", shellQuote(dest), shellQuote(url)))
}
