// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestStatusCode(t *testing.T) {
	statuses := []Status{StatusUnknown, StatusHealthy, StatusUnhealthy}
	for _, stat := range statuses {
		t.Run(stat.String(), func(t *testing.T) {
			assert.Equal(t, stat, StatusCode(stat.StatusCode()))
		})
	}
}

func TestStatus_StatusCode(t *testing.T) {
	tests := map[Status]int{
		StatusHealthy:   http.StatusOK,
		StatusUnhealthy: http.StatusServiceUnavailable,
		StatusUnknown:   http.StatusTooEarly,
		-3:              http.StatusInternalServerError,
	}
	for stat, want := range tests {
		assert.Equal(t, want, stat.StatusCode())
	}
}

func TestStatus_ExitCode(t *testing.T) {
	tests := map[Status]int{
		StatusHealthy:   0,
		StatusUnhealthy: int(StatusUnhealthy),
		StatusUnknown:   100,
		-3:              -3,
	}
	for stat, want := range tests {
		assert.Equal(t, want, stat.ExitCode())
	}
}

func TestCombine(t *testing.T) {
	tests := map[string]struct {
		a, b, want Status
	}{
		"both unknown":        {StatusUnknown, StatusUnknown, StatusUnknown},
		"both healthy":        {StatusHealthy, StatusHealthy, StatusHealthy},
		"both unhealthy":      {StatusUnhealthy, StatusUnhealthy, StatusUnhealthy},
		"unknown + healthy":   {StatusUnknown, StatusHealthy, StatusHealthy},
		"unknown + unhealthy": {StatusUnknown, StatusUnhealthy, StatusUnhealthy},
		"healthy + unknown":   {StatusHealthy, StatusUnknown, StatusUnhealthy},
		"healthy + unhealthy": {StatusHealthy, StatusUnhealthy, StatusUnhealthy},
		"unhealthy + unknown": {StatusUnhealthy, StatusUnknown, StatusUnhealthy},
		"unhealthy + healthy": {StatusUnhealthy, StatusHealthy, StatusUnhealthy},
		"invalid":             {3, -3, StatusUnknown},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, Combine(tc.a, tc.b))
		})
	}
}
