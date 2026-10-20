package configurer

type Configurer interface {
	Kind() string
	OSKind() string
	K3sCmdf(template string, args ...interface{}) string
	K3sBinaryPath() string
	K3sConfigPath() string
	K3sConfigDirPath() string
	K3sDataDirDefaultPath() string
	K3sTokenPath() string
	UpdateEnvironment(h Host, env map[string]string) error
	KubectlCmdf(h Host, dataDir string, s string, args ...interface{}) string
	KubeconfigPath(h Host, dataDir string) string
	FixContainer(h Host) error
	PrivateInterface(h Host) (string, error)
	PrivateAddress(h Host, iface, publicIP string) (string, error)
	K3sctlLockFilePath() string
	SetPath(key, value string)
}

type HostValidator interface {
	ValidateHost(h Host) error
}

type Preparable interface {
	Prepare(h Host) error
}
