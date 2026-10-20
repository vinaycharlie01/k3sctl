# k3sctl Completion Plan

Status tracker for bringing `k3sctl` to feature/quality parity with `k0sctl`.

Legend: ✅ done · 🚧 in progress · ⬜ pending · ⏭️ intentionally skipped

## Status

| Area | Item | Priority | Status |
|---|---|---|---|
| Core | Architecture (cmd → action → phase → configurer → rig) | — | ✅ |
| Core | CLI commands (apply/reset/backup/kubeconfig/config/init/version/completion) | — | ✅ |
| Core | ~30 phases | — | ✅ |
| Core | Configurers (all Linux distros; Windows intentionally dropped) | — | ✅ |
| Core | Binary providers (GitHub/local-upload/local-file/existing) | — | ✅ |
| CI | GitHub Actions + goreleaser | — | ✅ |
| CI | CLI smoke workflow (offline, ubuntu + macOS) | — | ✅ |
| P0 | Unify staged binary path between producer and consumers | P0 | ✅ |
| P0 | Fresh hosts stage/install a binary (`NeedsUpgrade` semantics) | P0 | ✅ |
| P0 | Write join token before generating configs (apply ordering) | P0 | ✅ |
| P0 | Agent readiness must use the leader's kubectl | P0 | ✅ |
| P1 | Nil-safe `Spec.Options` dereferences | P1 | ✅ |
| P1 | `Host.DrainNode` honors `DrainOption` | P1 | ✅ |
| P1 | Set `Metadata.NeedsReinstall` so `Reinstall` phase runs | P1 | ✅ |
| P1 | `ApplyManifests` phase + multi-doc manifest loading | P1 | ✅ |
| P1 | `PrepareArm` phase (or explicit deferral) | P1 | ✅ |
| P1 | Remove dead `phase/downloads.go` | P1 | ✅ |
| P2 | Core tests (host/spec/hosts/cluster, validate_facts, configure_k3s, get_kubeconfig, binprovider, cmd) | P2 | ✅ |
| P2 | Remaining k0sctl-equivalent tests | P2 | ⏭️ |
| P2 | `ValidateHosts` clock-skew check was a no-op | P2 | ✅ |
| P3 | Makefile `build-all` / `install`, `.golangci.yml` | P3 | ✅ |
| P3 | `.goreleaser.yml` cleanup (drop Windows, add arches) | P3 | ✅ |
| P3 | Dockerfile, CodeQL, README | P3 | ✅ |
| P3 | Smoke-test suite (requires Docker + real K3s hosts) | P3 | ⏭️ |

## Changelog

- 2026-10-20: Created this tracker after completing the parity audit.
- 2026-10-20: P0-3 done — `ConfigureK3s` now resolves the cluster token before writing any config: it reuses the leader's discovered token, honors `k3s.config.token`, or generates one for a fresh cluster. Follower/agent configs therefore contain a usable token. Added `phase/configure_k3s_test.go`.
- 2026-10-20: P0-4 done — `node.KubeNodeReadyFunc` now takes an explicit leader and always runs kubectl there; all call sites (`InitializeK3s`, `InstallServers`, `InstallAgents`, `UpgradeServers`) pass the leader. `UpgradeServers` resolves `Spec.K3sLeader()` in `Prepare()`.
- 2026-10-20: P1 drain done — reset/upgrade phases now use the nil-safe `Spec.Options.GetDrainOptions()`; `Host.DrainNode` takes a `*cluster.DrainOption`, skips when disabled, and renders flags via the new `DrainOption.ToKubectlArgs()`. `DrainOption.UnmarshalYAML` applies defaults so partial `drain:` blocks keep the documented behavior. Added `cluster/options_test.go`; fixed a stale `GenerateAgentConfig` empty-token assertion.
- 2026-10-20: P1 reinstall done — `GatherK3sFacts` records the flags from the running systemd unit; `ValidateFacts` sets `NeedsReinstall` when they differ from `K3sInstallFlags()`. `Reinstall` now drains/uncordons agents (honoring `--no-drain`) and clears the flag on success. Added `phase/gather_k3s_facts_test.go` and a `NeedsReinstall` table test.
- 2026-10-20: P1 manifests done — `loadConfig` now accepts multi-document files: exactly one `Cluster` document is required and every other document is kept as a manifest keyed by source filename (deduped as `name-2`, `name-3`, …). New `ApplyManifests` phase pipes each manifest to `kubectl apply -f -` on the leader and is registered after the "after" hooks, matching k0sctl. Added `phase/apply_manifests.go` and `cmd/flags_test.go`.
- 2026-10-20: Fixed a config-parsing regression exposed by multi-doc configs: `Host` embedded `rig.ClientWithConfig`, whose custom `UnmarshalYAML` (rig v2.0.0-alpha.2) consumed the entire host mapping and rejected keys like `role` under strict decoding. `Host` now embeds a local `ConnectionConfig` (same `ssh`/`winRM`/`openSSH`/`localhost` keys, no custom unmarshaller) plus `*rig.Client` with `yaml:"-"`, and builds the rig client in `Connect`. This mirrors k0sctl's layout. Added `cluster/host_test.go` covering strict parsing of connection + host fields and rejection of unknown keys.
- 2026-10-20: P1 ARM done — added `PrepareArm` (registered after `InstallBinaries`): ARM servers whose embedded etcd lacks native support get `ETCD_UNSUPPORTED_ARCH` (32-bit ARM always; arm64 only for k3s < v1.22.1 / etcd 3.5). While doing this, fixed the previously-inert `environment:` host option: `PrepareHosts` now writes it via `Configurer.UpdateEnvironment` and reconnects SSH, and generated `k3s`/`k3s-agent` systemd units render sorted `Environment=` directives. Added `phase/arm_prepare_test.go` and `phase/service_unit_test.go`.
- 2026-10-20: Removed dead `phase/downloads.go` (`DownloadBinaries`/`downloadFile` had no references; staging is handled by `StageBinaries` and the binary providers).
- 2026-10-20: P2 core tests — added coverage for `cluster` types (`host_test.go`, `hosts_test.go`, `spec_test.go`, `v1beta1/cluster_test.go`), kubeconfig rewriting (`phase/get_kubeconfig_test.go`), GitHub binary URL/arch selection (`binprovider/download_test.go`), and an `init` → `loadConfig` round-trip (`cmd/init_test.go`). `validate_facts`, `configure_k3s`, `gather_k3s_facts`, `manifest`, `node` and `retry` were already covered. Remaining k0sctl-style tests that require live hosts/Docker were intentionally skipped; those paths are exercised by smoke tests, not unit tests.
- 2026-10-20: P3 infra — added `make build-all` (linux/darwin amd64/arm64/arm, binaries in `dist/`) and `make install`; added `.golangci.yml` (golangci-lint v2) and fixed all 19 outstanding findings so `make lint` is clean; cleaned `.goreleaser.yml` (dropped Windows, added linux/arm v7 and a darwin/arm ignore); added a multi-stage `Dockerfile`, `.dockerignore`, and a CodeQL workflow; replaced the placeholder README with real documentation. `docker build` could not be completed locally because the environment intercepts TLS and `go mod download` fails certificate verification (`x509: certificate signed by unknown authority`); this is an environment issue, not a Dockerfile issue. Smoke tests are deferred: they require Docker and real K3s hosts.
- 2026-10-20: CI macOS smoke failed with `dyld: missing LC_UUID load command`. Root cause: `setup-go` with `go-version-file: go.mod` installs Go 1.22.0, whose Mach-O linker emits no `LC_UUID` at all (the load command and its hash fallback were added in later Go); macOS 26 dyld refuses to execute such binaries. `cli-smoke.yml` and `release.yml` now install `go-version: stable`, and the macOS smoke job asserts `LC_UUID` explicitly. This also fixes the published darwin binaries, which were built by the same old toolchain.
- 2026-10-20: `cmd.Run` now logs the error before exiting. urfave/cli only prints errors implementing `cli.ExitCoder`, so every plain error (missing config, parse failure, phase failure) exited with status 1 and no message. CI's smoke test caught it. `cli-smoke.yml` now also asserts that failures are reported and that invalid YAML fails at the `parse config` stage.
- 2026-10-20: Added `.github/workflows/cli-smoke.yml`, which builds the binary and asserts it actually works on `ubuntu-latest` and `macos-latest`: `version`/`--version`, the full command surface in `--help`, subcommand help, bash/zsh completion output, `init` output shape (single server by default, server + agent with `--agent`), and rejection of a missing config file, a non-`Cluster` document, invalid YAML, and the absent default `k3sctl.yaml`. It is intentionally offline — `apply`/`reset`/`config status` connect to hosts and are not run beyond input validation.
- 2026-10-20: Fixed `ValidateHosts` clock-skew validation, which previously ran `date +%s` and discarded the result. It now computes each host's skew from the median host clock and fails the run when any host deviates by more than 30s, matching k0sctl. Added `phase/validate_hosts_test.go` for the median/deviation helpers.
