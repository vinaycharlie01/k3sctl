# Roadmap: Phase Manager & Generic Phase

## Source Files (k0sctl)

### phase/manager.go
- `Manager` struct: phases []Phase, Config *v1beta1.Cluster, Concurrency int, ConcurrentUploads int, DryRun bool, Writer io.Writer, dryMessages map[string][]string
- Interfaces defined:
  - `Phase`: `Run(ctx) error`, `Title() string`
  - `withconfig`: `Prepare(config)` — receives config reference
  - `conditional`: `ShouldRun() bool` — skip logic
  - `withcleanup`: `CleanUp()` — cleanup on failure
  - `withmanager`: `SetManager(*Manager)`
  - `withDryRun`: `DryRun(io.Writer)` — dry-run alternative
  - `withBefore`: `Before(ctx) error` — pre-run hook
  - `withAfter`: `After(ctx) error` — post-run hook
- `Run()` method iterates phases:
  1. SetManager
  2. Prepare (receives config)
  3. ShouldRun check
  4. Before hook
  5. Run (or DryRun)
  6. After hook
- On error: runs CleanUp on already-executed phases in reverse
- Dry-run mode: collects messages per host, prints summary
- **k3sctl**: Port as-is with k3sctl types. This is infrastructure code, not k0s-specific.

### phase/manager_test.go
- Tests for phase execution order, conditional skipping, cleanup on failure, dry-run
- **k3sctl**: Port directly

### phase/generic_phase.go
- `GenericPhase` struct: embeds `*v1beta1.Cluster` config, `*Manager` pointer
- Provides:
  - `GetConfig() *v1beta1.Cluster`
  - `Prepare(config)` — stores config reference
  - `Wet(host, dryMsg, funcs...)` — run funcs only if not dry-run, otherwise DryMsg
  - `IsWet() bool` — not in dry-run mode
  - `DryMsg(host, msg)` — record dry-run message
  - `DryMsgf(host, format, args...)` — formatted dry-run message
  - `SetManager(m *Manager)`
  - `parallelDo(ctx, hosts, fn)` — parallel execution respecting `Manager.Concurrency`
  - `parallelDoUpload(ctx, hosts, fn)` — parallel execution respecting `Manager.ConcurrentUploads`
  - `runHooks(ctx, host, action, stage)` — execute hooks defined in host config
- **k3sctl**: Port as-is. This is infrastructure code.

## Key Design Patterns

1. **Phase lifecycle**: SetManager → Prepare → ShouldRun → Before → Run → After → CleanUp
2. **Wet/Dry pattern**: `Wet()` guards all mutations for dry-run support
3. **Parallel execution**: `parallelDo` uses semaphore pattern with configurable concurrency
4. **Upload throttling**: Separate limit for file uploads via `parallelDoUpload`
5. **Error cleanup**: On failure, CleanUp runs in reverse phase order

## Implementation Order
1. `phase/manager.go` + interfaces (direct port)
2. `phase/generic_phase.go` (direct port)
3. `phase/manager_test.go` (direct port)
