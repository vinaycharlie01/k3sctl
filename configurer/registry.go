package configurer

import (
	"fmt"
	"sync"

	"log/slog"

	rigos "github.com/k0sproject/rig/v2/os"
)

type osModuleEntry struct {
	matcher func(r *rigos.Release) bool
	builder func() any
}

var (
	osModules   []osModuleEntry
	osModulesMu sync.Mutex
)

func RegisterOSModule(matcher func(r *rigos.Release) bool, builder func() any) {
	osModulesMu.Lock()
	defer osModulesMu.Unlock()
	osModules = append(osModules, osModuleEntry{matcher: matcher, builder: builder})
}

func ResolveOSModule(r *rigos.Release) (Configurer, error) {
	osModulesMu.Lock()
	defer osModulesMu.Unlock()

	for _, m := range osModules {
		if m.matcher(r) {
			c, ok := m.builder().(Configurer)
			if !ok {
				return nil, fmt.Errorf("OS module for %s does not implement Configurer", r.ID)
			}
			slog.Debug(fmt.Sprintf("resolved OS module for %s: %T", r.ID, c))
			return c, nil
		}
	}

	for _, idLike := range r.IDLike {
		for _, m := range osModules {
			fakeRelease := &rigos.Release{ID: idLike}
			if m.matcher(fakeRelease) {
				c, ok := m.builder().(Configurer)
				if !ok {
					continue
				}
				slog.Debug(fmt.Sprintf("resolved OS module for %s via ID_LIKE=%s: %T", r.ID, idLike, c))
				return c, nil
			}
		}
	}

	return nil, fmt.Errorf("unsupported OS: %s (ID_LIKE: %v)", r.ID, r.IDLike)
}
