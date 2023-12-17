// Copyright (c) 2023, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
)

var okBytes = []byte(ok)

// HttpHandler is the http.Handler that writes a default ok message.
func HttpHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(okBytes)
	})
}

func (c *Config) Handler() http.Handler { return HttpHandler() }

type StatusChecker interface {
	CheckStatus(ctx context.Context) Status
}

type StatusCheckerFunc func(ctx context.Context) Status

func (fn StatusCheckerFunc) CheckStatus(ctx context.Context) Status { return fn(ctx) }

var _ http.Handler = new(Server)

type Server struct {
	config *Config
	checks map[string]StatusChecker
	status Status
}

func (c *Config) Server() *Server {
	return &Server{
		config: c,
		checks: make(map[string]StatusChecker, 2),
		status: Unknown,
	}
}

func (s *Server) Check(name string, check StatusChecker) { s.checks[name] = check }

func (s *Server) CheckFunc(name string, check StatusCheckerFunc) { s.Check(name, check) }

func (s *Server) ServeHTTP(wri http.ResponseWriter, req *http.Request) {
	n := len(s.checks)
	if n == 0 {
		s.setStatus(Healthy)
		_, _ = wri.Write(okBytes)
		return
	}

	ctx, cancel := timeoutContext(req.Context(), s.config.Timeout)
	defer cancel()

	var stat Status
	result := make(map[string]string, n)

	if s.config.CheckParallel {
		cp := checkParallel{result: result}
		for name, check := range s.checks {
			cp.do(ctx, name, check)
		}
		cp.Wait()
		stat = cp.status
	} else {
		for name, check := range s.checks {
			result[name] = updateStatus(&stat, check.CheckStatus(ctx)).String()
		}
	}

	wri.Header().Set(s.config.Header, stat.String())
	s.setStatus(stat)

	if b, err := json.Marshal(result); err != nil {
		wri.WriteHeader(http.StatusInternalServerError)
	} else {
		wri.WriteHeader(stat.StatusCode())
		wri.Header().Set("Content-Type", "application/json")
		_, _ = wri.Write(b)
	}
}

func (s *Server) setStatus(stat Status) {
	if s.config.Listener != nil && stat == Healthy && s.status != Healthy {
		s.config.Listener.HealthChanged(stat, s.status)
	}

	s.status = stat
}

type checkParallel struct {
	sync.WaitGroup
	mut sync.Mutex

	result map[string]string
	status Status
}

func (cp *checkParallel) do(ctx context.Context, name string, check StatusChecker) {
	cp.Add(1)
	go func() {
		stat := check.CheckStatus(ctx)

		cp.mut.Lock()
		cp.result[name] = updateStatus(&cp.status, stat).String()
		cp.mut.Unlock()
		cp.Done()
	}()
}

func updateStatus(dest *Status, stat Status) Status {
	if *dest == Error {
		return stat
	}
	if stat == Error {
		*dest = Error
	} else if *dest == Healthy {
		*dest = stat
	}

	return stat
}
