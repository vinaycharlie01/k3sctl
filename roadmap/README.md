# k3sctl Development Roadmap

Complete file-by-file analysis of k0sctl adapted for K3s.

## Documents

| File | Contents |
|------|----------|
| [00-OVERVIEW.md](00-OVERVIEW.md) | Project overview, K0s vs K3s differences, dependencies |
| [01-FILE-INVENTORY.md](01-FILE-INVENTORY.md) | Every k0sctl file mapped to k3sctl with effort rating |
| [02-CMD-ACTION-ANALYSIS.md](02-CMD-ACTION-ANALYSIS.md) | CLI commands and action workflows |
| [03-PHASE-ANALYSIS.md](03-PHASE-ANALYSIS.md) | All phase implementations |
| [04-CONFIGURER-ANALYSIS.md](04-CONFIGURER-ANALYSIS.md) | OS-specific configurers |
| [05-API-TYPES-ANALYSIS.md](05-API-TYPES-ANALYSIS.md) | API types, binary providers, supporting packages |
| [06-IMPLEMENTATION-PLAN.md](06-IMPLEMENTATION-PLAN.md) | Phased development plan with file counts |
| [07-K3S-SPECIFICS.md](07-K3S-SPECIFICS.md) | K3s technical details for implementation |

## Quick Stats

- **~90 Go source files** in k0sctl analyzed
- **~130 files** estimated for k3sctl
- **~25 direct ports** (no K3s-specific changes)
- **~30 moderate adaptations** (rename + path changes)
- **~30 major rewrites** (K3s-specific logic)
- **2-3 dropped** (K3s N/A)
- **2-3 new** (K3s-specific additions)
