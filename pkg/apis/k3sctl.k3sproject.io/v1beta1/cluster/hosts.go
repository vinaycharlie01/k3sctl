package cluster

import (
	"context"
	"fmt"
	"sync"
)

type Hosts []*Host

func (hosts Hosts) Filter(fn func(h *Host) bool) Hosts {
	var result Hosts
	for _, h := range hosts {
		if fn(h) {
			result = append(result, h)
		}
	}
	return result
}

func (hosts Hosts) Servers() Hosts {
	return hosts.Filter(func(h *Host) bool { return h.IsServer() })
}

func (hosts Hosts) Agents() Hosts {
	return hosts.Filter(func(h *Host) bool { return h.IsAgent() })
}

func (hosts Hosts) First() *Host {
	if len(hosts) == 0 {
		return nil
	}
	return hosts[0]
}

func (hosts Hosts) Leader() *Host {
	servers := hosts.Servers()
	for _, h := range servers {
		if h.Metadata.IsLeader {
			return h
		}
	}
	return servers.First()
}

func (hosts Hosts) Find(fn func(h *Host) bool) *Host {
	for _, h := range hosts {
		if fn(h) {
			return h
		}
	}
	return nil
}

func (hosts Hosts) Each(fn func(h *Host) error) error {
	for _, h := range hosts {
		if err := fn(h); err != nil {
			return err
		}
	}
	return nil
}

func (hosts Hosts) ParallelEach(ctx context.Context, concurrency int, fn func(ctx context.Context, h *Host) error) error {
	if concurrency <= 0 {
		concurrency = len(hosts)
	}

	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex
	var firstErr error

	var wg sync.WaitGroup
	for _, h := range hosts {
		wg.Add(1)
		go func(host *Host) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := fn(ctx, host); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("%s: %w", host, err)
				}
				mu.Unlock()
			}
		}(h)
	}
	wg.Wait()
	return firstErr
}

func (hosts Hosts) BatchedParallelEach(ctx context.Context, concurrency, batchSize int, fn func(ctx context.Context, h *Host) error) error {
	if batchSize <= 0 || batchSize > len(hosts) {
		batchSize = len(hosts)
	}

	for i := 0; i < len(hosts); i += batchSize {
		end := i + batchSize
		if end > len(hosts) {
			end = len(hosts)
		}
		batch := hosts[i:end]
		if err := batch.ParallelEach(ctx, concurrency, fn); err != nil {
			return err
		}
	}
	return nil
}

func (hosts Hosts) Count() int {
	return len(hosts)
}

func (hosts Hosts) Addresses() []string {
	addrs := make([]string, 0, len(hosts))
	for _, h := range hosts {
		addrs = append(addrs, h.Address())
	}
	return addrs
}
