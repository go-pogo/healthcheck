// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithLogger(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		c := Checker{log: DefaultLogger(nil)}
		assert.NoError(t, WithLogger(nil)(&c))
		assert.Nil(t, c.log)
	})

	t.Run("non-nil", func(t *testing.T) {
		var c Checker
		want := NopLogger()
		assert.NoError(t, WithLogger(want)(&c))
		assert.Same(t, want, c.log)
	})
}

func TestWithHealthChecker(t *testing.T) {
	t.Run("nil checker", func(t *testing.T) {
		assert.PanicsWithValue(t, panicNilHealthChecker, func() {
			_ = WithHealthChecker("test", nil)
		})
	})

	t.Run("register", func(t *testing.T) {
		const name = "test"
		want := new(alwaysHealty)

		var c Checker
		assert.Nil(t, c.checks)
		assert.NoError(t, WithHealthChecker(name, want)(&c))
		assert.Len(t, c.checks, 1)
		assert.Same(t, want, c.checks[name])
	})
}

var _ HealthChecker = (*alwaysHealty)(nil)

type alwaysHealty struct{}

func (alwaysHealty) CheckHealth(context.Context) Status { return StatusHealthy }
