// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

type Option func(c *Checker) error

const panicNilLogger = "healthcheck.WithLogger: Logger should not be nil"

func WithLogger(log Logger) Option {
	return func(c *Checker) error {
		if log == nil {
			panic(panicNilLogger)
		}

		c.log = log
		return nil
	}
}

func WithDefaultLogger() Option { return WithLogger(DefaultLogger()) }

func WithHealthChecker(name string, check HealthChecker) Option {
	if check == nil {
		panic(panicNilHealthChecker)
	}

	return func(c *Checker) error {
		c.register(name, check)
		return nil
	}
}
