// Package environment defines the provisioning abstraction shared by the
// integration test runners. A concrete implementation (for example the LXD
// provider) owns host lifecycle, while the test runner owns SSH provisioning
// and assertions.
package environment

import "context"

// Role is the K3s role of a node.
type Role string

const (
	RoleServer Role = "server"
	RoleAgent  Role = "agent"
)

// Node describes a provisioned host.
type Node struct {
	// Name is the unique, run-scoped instance name. It is also used as the
	// hostname inside the guest and therefore as the Kubernetes node name.
	Name string
	// Role is the K3s role the node will be assigned.
	Role Role
	// Address is the IPv4 address reachable from the test process.
	Address string
	// SSHUser and SSHPort describe how the test runner connects to the node.
	SSHUser string
	SSHPort int
}

// Environment provisions and tears down the hosts used by an integration test.
//
// Create must be safe to call after a partial failure: a later Destroy must
// remove exactly the resources created by this environment, and nothing else.
//
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.14.0 --fake-name FakeEnvironment -o ./mocks/fake_environment.go . Environment
type Environment interface {
	Create(ctx context.Context) ([]Node, error)
	Bootstrap(ctx context.Context, nodes []Node) error
	Destroy(ctx context.Context) error
}

// DiagnosticProvider is implemented by environments that can report
// platform-level state for failure diagnostics.
//
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.14.0 --fake-name FakeDiagnosticProvider -o ./mocks/fake_diagnostic_provider.go . DiagnosticProvider
type DiagnosticProvider interface {
	Diagnostics(ctx context.Context, nodes []Node) (string, error)
}
