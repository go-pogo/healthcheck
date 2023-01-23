// Copyright (c) 2023, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

type Logger interface {
	Healthy()
}

// NopLogger is a Logger that does not log anything.
func NopLogger() Logger { return &nopLogger{} }

type nopLogger struct{}

func (*nopLogger) Healthy() {}
