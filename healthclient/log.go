// Copyright (c) 2025, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthclient

import (
	"log"

	"github.com/go-pogo/healthcheck"
)

type Logger interface {
	LogHealthChecked(stat healthcheck.Status)
	LogHealthCheckFailed(stat healthcheck.Status, err error)
}

const panicNewNilLogger = "healthcheck.NewLogger: log.Logger should not be nil"

// NewLogger returns a [Logger] that uses a [log.Logger] to log health
// status events.
func NewLogger(l *log.Logger) Logger {
	if l == nil {
		panic(panicNewNilLogger)
	}
	return &logger{l}
}

// DefaultLogger returns a [Logger] that uses [log.Default] to log health
// status events.
func DefaultLogger() Logger { return &logger{log.Default()} }

// NopLogger returns a [Logger] that does nothing.
func NopLogger() Logger { return new(nopLogger) }

type logger struct{ *log.Logger }

func (l *logger) LogHealthChecked(stat healthcheck.Status) {
	l.Printf("health checked: %s\n", stat)
}

func (l *logger) LogHealthCheckFailed(stat healthcheck.Status, err error) {
	l.Printf("health check failed: %s: %+v\n", stat, err)
}

type nopLogger struct{}

func (*nopLogger) LogHealthChecked(_ healthcheck.Status) {}

func (*nopLogger) LogHealthCheckFailed(_ healthcheck.Status, _ error) {}
