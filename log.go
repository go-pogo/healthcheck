// Copyright (c) 2023, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

import "log"

type Logger interface {
	Notifier
	HealthChecked(name string, stat Status)
}

// DefaultLogger returns a [Logger] that uses a [log.Logger] to log health
// status events. It defaults to [log.Default] if the provided [log.Logger] l
// is nil.
func DefaultLogger(l *log.Logger) Logger {
	if l == nil {
		l = log.Default()
	}
	return &logger{l}
}

// NopLogger returns a [Logger] that does nothing.
func NopLogger() Logger { return new(nopLogger) }

type logger struct{ *log.Logger }

func (l *logger) HealthChanged(status, oldStatus Status) {
	l.Logger.Println("health changed from " + oldStatus.String() + " to " + status.String())
}

func (l *logger) HealthChecked(name string, stat Status) {
	l.Logger.Printf("health for %s is %s\n", name, stat)
}

type nopLogger struct{}

func (*nopLogger) HealthChanged(_, _ Status)    {}
func (*nopLogger) HealthChecked(string, Status) {}
