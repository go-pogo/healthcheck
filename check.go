// Copyright (c) 2022, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

import (
	"context"
	"sync"
	"time"
)

type Notifier interface {
	StatusChanged(status, oldStatus Status)
}

type StatusChecker interface {
	CheckStatus(ctx context.Context) Status
}

type StatusCheckerFunc func(ctx context.Context) Status

func (fn StatusCheckerFunc) CheckStatus(ctx context.Context) Status { return fn(ctx) }

type CheckerConfig struct {
	Timeout  time.Duration
	Parallel bool
}

type Checker struct {
	CheckerConfig

	//log      Logger
	mut      sync.RWMutex
	checks   map[string]StatusChecker
	statuses map[string]Status
	status   AtomicStatus
}

func NewChecker(conf CheckerConfig) *Checker {
	return &Checker{
		CheckerConfig: conf,
		checks:        make(map[string]StatusChecker),
		statuses:      make(map[string]Status),
	}
}

func (h *Checker) Status() Status { return h.status.Load() }

func (h *Checker) Statuses() map[string]Status {
	h.mut.RLock()
	defer h.mut.RUnlock()
	return h.statuses
}

func (h *Checker) Add(name string, check StatusChecker) {
	h.mut.Lock()
	h.checks[name] = check
	h.mut.Unlock()
}

func (h *Checker) AddFunc(name string, check StatusCheckerFunc) {
	h.Add(name, check)
}

func (h *Checker) Remove(name string) {
	h.mut.Lock()
	delete(h.checks, name)
	h.mut.Unlock()
}

func (h *Checker) CheckStatus(ctx context.Context) Status {
	h.mut.RLock()
	defer h.mut.RUnlock()

	if len(h.checks) == 0 {
		defer h.mut.RUnlock()
		h.setStatus(StatusHealthy)
		return StatusHealthy
	}

	timeout := h.CheckerConfig.Timeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}
	if t, ok := ctx.Deadline(); !ok || timeout < time.Until(t) {
		var cancelFn context.CancelFunc
		ctx, cancelFn = context.WithTimeout(ctx, timeout)
		defer cancelFn()
	}

	var status AtomicStatus
	checkStatus := func(stat Status) {
		switch stat {
		case StatusHealthy:
			if status.Load() != StatusUnhealthy {
				status.Store(stat)
			}
		case StatusUnhealthy:
			status.Store(stat)
		}
	}

	if len(h.checks) == 1 || !h.CheckerConfig.Parallel {
		for name, c := range h.checks {
			stat := c.CheckStatus(ctx)
			//h.log.StatusChecked(name, stat)
			checkStatus(stat)
		}
	} else {
		var wg sync.WaitGroup
		wg.Add(len(h.checks))

		for name, c := range h.checks {
			go func(name string, c StatusChecker) {
				defer wg.Done()
				stat := c.CheckStatus(ctx)
				//h.log.StatusChecked(name, stat)
				checkStatus(stat)
			}(name, c)
		}
		wg.Wait()
	}

	stat := status.Load()
	h.setStatus(stat)
	return stat
}

func (h *Checker) setStatus(stat Status) {
	old := h.status.Swap(stat)
	if old == stat {
		return
	}
	//if h.log != nil {
	//	h.log.StatusChanged(stat, old)
	//}
	// notify other listeners?
}
