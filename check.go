// Copyright (c) 2022, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

import (
	"context"
	"github.com/go-pogo/errors"
	"sync"
	"time"
)

// HealthChecker checks the status of a service.
type HealthChecker interface {
	CheckHealth(ctx context.Context) Status
}

// HealthCheckerFunc checks the status of a service.
type HealthCheckerFunc func(ctx context.Context) Status

func (fn HealthCheckerFunc) CheckHealth(ctx context.Context) Status { return fn(ctx) }

// Registerer registers [HealthChecker](s).
type Registerer interface {
	Register(name string, check HealthChecker)
}

// HealthCheckerRegisterer registers [HealthChecker](s) to a [Registerer].
type HealthCheckerRegisterer interface {
	RegisterHealthCheckers(r Registerer)
}

var (
	_ Registerer    = (*Checker)(nil)
	_ HealthChecker = (*Checker)(nil)
)

type Checker struct {
	Timeout time.Duration
	// Parallel indicates whether to run health checks in parallel.
	Parallel bool

	log      Logger
	mut      sync.RWMutex
	checks   map[string]HealthChecker
	statuses map[string]Status
	status   AtomicStatus
}

func New(opts ...Option) (*Checker, error) {
	c := Checker{Timeout: 3 * time.Second}
	if err := c.with(opts); err != nil {
		return nil, err
	}
	if len(c.checks) > 2 {
		c.Parallel = true
	}
	if c.log == nil {
		c.log = NopLogger()
	}
	return &c, nil
}

func (h *Checker) With(opts ...Option) error {
	h.mut.Lock()
	defer h.mut.Unlock()
	return h.with(opts)
}

func (h *Checker) with(opts []Option) error {
	var err error
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		err = errors.Append(err, opt(h))
	}
	return nil

}

// Status returns the current health [Status] based on the statuses of all
// registered [HealthChecker](s).
func (h *Checker) Status() Status { return h.status.Load() }

// Statuses returns a map of the statuses of all registered [HealthChecker](s).
func (h *Checker) Statuses() map[string]Status {
	h.mut.RLock()
	defer h.mut.RUnlock()
	return h.copyStatuses()
}

func (h *Checker) copyStatuses() map[string]Status {
	stats := make(map[string]Status, len(h.statuses))
	for k, v := range h.statuses {
		stats[k] = v
	}
	return stats
}

const panicNilHealthChecker = "healthcheck: HealthChecker should not be nil"

// Register a [HealthChecker] with the given name.
func (h *Checker) Register(name string, check HealthChecker) {
	if check == nil {
		panic(panicNilHealthChecker)
	}

	h.mut.Lock()
	h.register(name, check)
	h.mut.Unlock()
}

func (h *Checker) register(name string, check HealthChecker) {
	if h.checks == nil {
		h.checks = map[string]HealthChecker{name: check}
	} else {
		h.checks[name] = check
	}
}

// Unregister the [HealthChecker] with the given name.
func (h *Checker) Unregister(name string) {
	h.mut.Lock()
	if h.checks != nil {
		delete(h.checks, name)
	}
	h.mut.Unlock()
}

// CheckHealth triggers a health check for all registered [HealthChecker](s).
func (h *Checker) CheckHealth(ctx context.Context) Status {
	h.mut.RLock()
	if len(h.checks) == 0 {
		defer h.mut.RUnlock()
		h.setStatus(StatusHealthy)
		return StatusHealthy
	}

	h.mut.RUnlock()
	h.mut.Lock()
	defer h.mut.Unlock()

	if h.statuses == nil {
		h.statuses = make(map[string]Status, len(h.checks))
	}

	if h.Timeout > 0 {
		// add a timeout to the context if none is set
		if t, ok := ctx.Deadline(); !ok || h.Timeout < time.Until(t) {
			var cancelFn context.CancelFunc
			ctx, cancelFn = context.WithTimeout(ctx, h.Timeout)
			defer cancelFn()
		}
	}

	// check health status for each registered service
	if len(h.checks) == 1 || !h.Parallel {
		for name, c := range h.checks {
			h.statuses[name] = c.CheckHealth(ctx)
		}
	} else {
		var wg sync.WaitGroup
		wg.Add(len(h.checks))
		for name, c := range h.checks {
			go func(name string, c HealthChecker) {
				defer wg.Done()
				h.statuses[name] = c.CheckHealth(ctx)
			}(name, c)
		}
		wg.Wait()
	}

	result := StatusUnknown
	for _, stat := range h.statuses {
		result = Combine(result, stat)
		if result == StatusUnhealthy {
			break
		}
	}

	h.setStatus(result)
	return result
}

func (h *Checker) setStatus(stat Status) {
	if old := h.status.Swap(stat); old != stat {
		h.log.LogHealthChanged(stat, old, h.copyStatuses())
	}
}
