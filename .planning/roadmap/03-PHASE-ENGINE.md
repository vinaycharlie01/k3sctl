# 03: Phase Manager and Generic Phase

## Overview

The phase engine is the execution core. It runs an ordered list of phases through a lifecycle: SetManager → Prepare → ShouldRun → Before → Run → After → CleanUp. This is architecture-neutral — identical for k3sctl.

## Files to Create

### `phase/manager.go`
- **Source**: `k0sctl/phase/manager.go` (~292 lines)
- **Changes**: Only import path changes (`k0sctl` → `k3sctl`)
- **Key types and interfaces**:
  - `Phase` interface: `Run(context.Context) error` + `Title() string`
  - `Phases` type: slice with `Index`, `Remove`, `InsertAfter`, `InsertBefore`, `Replace`
  - `withconfig`: `Prepare(*v1beta1.Cluster) error`
  - `conditional`: `ShouldRun() bool`
  - `withcleanup`: `CleanUp()`
  - `withmanager`: `SetManager(*Manager)`
  - `withDryRun`: `DryRun() error`
  - `withBefore`: `Before() error`
  - `withAfter`: `After() error`
- **Manager struct**:
  - `phases Phases`
  - `Config *v1beta1.Cluster`
  - `Concurrency int`
  - `ConcurrentUploads int`
  - `DryRun bool`
  - `Writer io.Writer`
  - `dryMessages map[string][]string` + `dryMu sync.Mutex`
- **Manager methods**:
  - `NewManager(config) (*Manager, error)`
  - `AddPhase(p ...Phase)`
  - `SetPhases(p Phases)`
  - `DryMsg(host, msg)` — records dry-run messages
  - `Wet(host, msg, funcs...)` — runs func in wet mode, records in dry mode
  - `Run(ctx) error` — main execution loop
- **Global vars** (package-level):
  - `NoWait bool`
  - `Force bool`
  - `Colorize = aurora.NewAurora(false)`

### `phase/generic_phase.go`
- **Source**: `k0sctl/phase/generic_phase.go` (~95 lines)
- **Changes**: Only import path changes
- **GenericPhase struct**: Embeds `Config *v1beta1.Cluster` and `manager *Manager`
- **Methods**:
  - `GetConfig()` — accessor
  - `Prepare(c *v1beta1.Cluster) error` — sets Config
  - `Wet(host, msg, funcs...)` — delegates to manager
  - `IsWet() bool` — !manager.DryRun
  - `DryMsg(host, msg)` — delegates to manager
  - `DryMsgf(host, msg, args...)` — formatted dry message
  - `SetManager(m *Manager)` — stores manager ref
  - `parallelDo(ctx, hosts, funcs...)` — uses hosts.ParallelEach or BatchedParallelEach
  - `parallelDoUpload(ctx, hosts, funcs...)` — respects ConcurrentUploads
  - `runHooks(ctx, action, stage, hosts...)` — executes per-host hooks

### `phase/manager_test.go`
- **Source**: `k0sctl/phase/manager_test.go`
- **Changes**: Import paths only

## Execution Flow (Manager.Run)

```
for each phase:
  1. Check ctx.Err() → abort if canceled
  2. SetManager(m) if phase implements withmanager
  3. Prepare(config) if phase implements withconfig
  4. ShouldRun() if phase implements conditional → skip if false
  5. Before() if phase implements withBefore
  6. If DryRun and phase implements withDryRun → call DryRun() instead
  7. Run(ctx) → execute the phase
  8. After() if Run succeeded and phase implements withAfter
  9. On error: run CleanUp() on all previously-run phases (reverse order implicitly)
```

## Key Design Decisions (Preserved)

- Phases are composed in action files, not hardcoded in manager
- `parallelDo` respects `Manager.Concurrency` (0 = unlimited)
- `parallelDoUpload` respects `Manager.ConcurrentUploads`
- Dry-run is first-class: every mutation guarded by `Wet()`
- CleanUp runs on error for all phases that have run
- The Phases slice supports dynamic manipulation (Insert/Remove/Replace)
