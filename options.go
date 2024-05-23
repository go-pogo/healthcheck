// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

type Option func(c *Checker) error

func WithLogger(l Logger) Option {
	return func(c *Checker) error {
		c.log = l
		return nil
	}
}

func WithHealthChecker(name string, check HealthChecker) Option {
	return func(c *Checker) error {
		c.register(name, check)
		return nil
	}
}
