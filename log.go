// Copyright (c) 2023, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

import (
	"log"
)

type Logger interface {
	LogHealthChanged(newStatus, oldStatus Status, statuses map[string]Status)
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

func (l *logger) LogHealthChanged(status, oldStatus Status, details map[string]Status) {
	l.Logger.Printf("health changed from %s to %s\n", oldStatus, status)
	if details == nil {
		return
	}

	for name, stat := range details {
		l.Logger.Printf("health for %s is %s\n", name, stat)
	}
}

type nopLogger struct{}

func (*nopLogger) LogHealthChanged(_, _ Status, _ map[string]Status) {}
