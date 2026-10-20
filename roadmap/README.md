# k3sctl Development Roadmap

Complete file-by-file analysis of k0sctl adapted for K3s, with implementation plan.

## Overview & Differences

| File | Contents |
|------|----------|
| [00-OVERVIEW.md](00-OVERVIEW.md) | Project overview, K0s vs K3s differences, dependencies, file counts |

## Detailed Analysis (k0sctl file-by-file)

| File | Contents |
|------|----------|
| [01-FILE-INVENTORY.md](01-FILE-INVENTORY.md) | Every k0sctl file mapped to k3sctl with effort rating |
| [02-CMD-ACTION-ANALYSIS.md](02-CMD-ACTION-ANALYSIS.md) | CLI commands and action workflows |
| [03-PHASE-ANALYSIS.md](03-PHASE-ANALYSIS.md) | All phase implementations |
| [04-CONFIGURER-ANALYSIS.md](04-CONFIGURER-ANALYSIS.md) | OS-specific configurers |
| [05-API-TYPES-ANALYSIS.md](05-API-TYPES-ANALYSIS.md) | API types, binary providers, supporting packages |
| [07-K3S-SPECIFICS.md](07-K3S-SPECIFICS.md) | K3s technical details (install, token, paths, systemd) |

## Implementation Roadmap (by component)

| File | Contents |
|------|----------|
| [01-ENTRYPOINT-AND-CLI.md](01-ENTRYPOINT-AND-CLI.md) | Entry point, CLI layer: main.go, cmd/ |
| [02-ACTIONS.md](02-ACTIONS.md) | Action layer: apply, reset, backup, kubeconfig |
| [03-PHASE-MANAGER.md](03-PHASE-MANAGER.md) | Phase manager framework |
| [04-PHASES-INFRASTRUCTURE.md](04-PHASES-INFRASTRUCTURE.md) | Reusable phases: connect, detect_os, facts, hooks |
| [05-PHASES-K3S-SPECIFIC.md](05-PHASES-K3S-SPECIFIC.md) | K3s phases: install, upgrade, reset, backup |
| [06-CONFIGURER.md](06-CONFIGURER.md) | Configurer layer: interface, registry, Linux distros |
| [07-API-TYPES.md](07-API-TYPES.md) | Config data model: Cluster, Host, K3s, Spec |
| [08-PACKAGES.md](08-PACKAGES.md) | Supporting packages: retry, download, binprovider, node |

## Implementation Plan & File Mapping

| File | Contents |
|------|----------|
| [06-IMPLEMENTATION-PLAN.md](06-IMPLEMENTATION-PLAN.md) | Overall phased plan from analysis |
| [09-FILE-MAPPING.md](09-FILE-MAPPING.md) | Complete k0sctl→k3sctl file mapping table |
| [10-IMPLEMENTATION-PLAN.md](10-IMPLEMENTATION-PLAN.md) | Detailed implementation plan with task checklist |

## Quick Stats

- **~132 Go files** in k0sctl analyzed (99 source + 33 test)
- **~109 files** estimated for k3sctl (82 source + 27 test)
- **~40 direct ports** (minimal changes)
- **~60 adaptations** (K3s-specific logic)
- **~3 new files** (K3s additions)
- **~6 omitted** (Windows support, join token workaround)
