package configurer

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.14.0 --fake-name FakeConfigurer -o ./mocks/fake_configurer.go . Configurer
type Configurer interface {
	Kind() string
	OSKind() string
	K3sCmdf(template string, args ...any) string
	K3sBinaryPath() string
	K3sConfigPath() string
	K3sConfigDirPath() string
	K3sDataDirDefaultPath() string
	K3sTokenPath() string
	UpdateEnvironment(h Host, env map[string]string) error
	KubectlCmdf(h Host, dataDir string, s string, args ...any) string
	KubeconfigPath(h Host, dataDir string) string
	FixContainer(h Host) error
	PrivateInterface(h Host) (string, error)
	PrivateAddress(h Host, iface, publicIP string) (string, error)
	K3sctlLockFilePath() string
	SetPath(key, value string)
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.14.0 --fake-name FakeHostValidator -o ./mocks/fake_host_validator.go . HostValidator
type HostValidator interface {
	ValidateHost(h Host) error
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.14.0 --fake-name FakePreparable -o ./mocks/fake_preparable.go . Preparable
type Preparable interface {
	Prepare(h Host) error
}
