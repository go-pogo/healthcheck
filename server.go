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

type StateChecker interface {
	CheckState(ctx context.Context) Status
}

type StateCheckerFunc func(ctx context.Context) Status

func (fn StateCheckerFunc) CheckState(ctx context.Context) Status { return fn(ctx) }

var _ http.Handler = new(Server)

type Server struct {
	config    *Config
	checks    map[string]StateChecker
	lastState Status
}

func (c *Config) Server() *Server {
	return &Server{
		config:    c,
		checks:    make(map[string]StateChecker, 2),
		lastState: Unknown,
	}
}

func (s *Server) Check(name string, check StateChecker) { s.checks[name] = check }

func (s *Server) CheckFunc(name string, check StateCheckerFunc) { s.Check(name, check) }

func (s *Server) ServeHTTP(wri http.ResponseWriter, req *http.Request) {
	n := len(s.checks)
	if n == 0 {
		s.logState(Healthy)
		_, _ = wri.Write(okBytes)
		return
	}

	ctx, cancel := timeoutContext(req.Context(), s.config.Timeout)
	defer cancel()

	var state Status
	result := make(map[string]string, n)

	if s.config.CheckParallel {
		cp := checkParallel{result: result}
		for name, check := range s.checks {
			cp.do(ctx, name, check)
		}
		cp.Wait()
		state = cp.state
	} else {
		for name, check := range s.checks {
			result[name] = updateState(&state, check.CheckState(ctx)).String()
		}
	}

	wri.Header().Set(s.config.Header, state.String())
	s.logState(state)

	if b, err := json.Marshal(result); err != nil {
		wri.WriteHeader(http.StatusInternalServerError)
		// log error?
	} else {
		wri.WriteHeader(state.StatusCode())
		wri.Header().Set("Content-Type", "application/json")
		_, _ = wri.Write(b)
	}
}

func (s *Server) logState(state Status) {
	if state == Healthy && s.lastState != Healthy {
		s.config.Logger.Healthy()
	}

	s.lastState = state
}

type checkParallel struct {
	sync.WaitGroup
	mut sync.Mutex

	result map[string]string
	state  Status
}

func (cp *checkParallel) do(ctx context.Context, name string, check StateChecker) {
	cp.Add(1)
	go func() {
		state := check.CheckState(ctx)

		cp.mut.Lock()
		cp.result[name] = updateState(&cp.state, state).String()
		cp.mut.Unlock()
		cp.Done()
	}()
}

func updateState(dest *Status, state Status) Status {
	if *dest == Error {
		return state
	}
	if state == Error {
		*dest = Error
	} else if *dest == Healthy {
		*dest = state
	}

	return state
}
