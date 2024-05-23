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

type Notifier interface {
	HealthChanged(status, oldStatus Status)
}

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
	var c Checker
	if err := c.with(opts); err != nil {
		return nil, err
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
	stats := h.statuses
	h.mut.RUnlock()
	return stats
}

// Register a [HealthChecker] with the given name.
func (h *Checker) Register(name string, check HealthChecker) {
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

	timeout := h.Timeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}
	if t, ok := ctx.Deadline(); !ok || timeout < time.Until(t) {
		var cancelFn context.CancelFunc
		ctx, cancelFn = context.WithTimeout(ctx, timeout)
		defer cancelFn()
	}

	var status AtomicStatus
	updateStatus := func(stat Status) {
		switch stat {
		case StatusHealthy:
			if status.Load() == StatusUnknown {
				status.Store(stat)
			}
		case StatusUnhealthy:
			status.Store(stat)

		case StatusUnknown:
			if status.Load() == StatusHealthy {
				status.Store(StatusUnhealthy)
			}
		}
	}

	if h.statuses == nil {
		h.statuses = make(map[string]Status, len(h.checks))
	}
	if len(h.checks) == 1 || !h.Parallel {
		for name, c := range h.checks {
			stat := c.CheckHealth(ctx)
			h.statuses[name] = stat
			h.log.HealthChecked(name, stat)
			updateStatus(stat)
		}
	} else {
		var wg sync.WaitGroup
		wg.Add(len(h.checks))

		for name, c := range h.checks {
			go func(name string, c HealthChecker) {
				defer wg.Done()
				stat := c.CheckHealth(ctx)
				h.statuses[name] = stat
				h.log.HealthChecked(name, stat)
				updateStatus(stat)
			}(name, c)
		}
		wg.Wait()
	}

	stat := status.Load()
	h.setStatus(stat)
	return stat
}

func (h *Checker) setStatus(stat Status) {
	if old := h.status.Swap(stat); old != stat {
		h.log.HealthChanged(stat, old)
	}
}
