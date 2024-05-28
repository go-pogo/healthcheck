// Copyright (c) 2022, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

import (
	"net/http"
	"strconv"
	"sync/atomic"
)

// Status describes the health status of a service.
// https://docs.docker.com/engine/reference/builder/#healthcheck
// https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/
type Status int8

const (
	// StatusUnknown indicates the service's status is yet to be determined.
	StatusUnknown Status = 0
	// StatusHealthy indicates the service is working as expected.
	StatusHealthy Status = 1
	// StatusUnhealthy indicates the service is not working correctly.
	StatusUnhealthy Status = -1
)

// StatusCode returns a [Status] representing the given http status code.
func StatusCode(c int) Status {
	switch c {
	case http.StatusTooEarly:
		return StatusUnknown
	case http.StatusOK:
		return StatusHealthy
	default:
		return StatusUnhealthy
	}
}

// StatusCode returns a http status code which represents [Status] in a
// [http.Response].
func (s Status) StatusCode() int {
	switch s {
	case StatusUnknown:
		return http.StatusTooEarly // 425
	case StatusHealthy:
		return http.StatusOK // 200
	case StatusUnhealthy:
		return http.StatusServiceUnavailable // 503
	default:
		return http.StatusInternalServerError // 500
	}
}

// ExitCode returns an exit code which can be used with [os.Exit].
func (s Status) ExitCode() int {
	switch s {
	case StatusHealthy:
		return 0
	case StatusUnknown:
		return 100
	default:
		return int(s)
	}
}

// String return a string representation of [Status].
func (s Status) String() string {
	switch s {
	case StatusHealthy:
		return "healthy"
	case StatusUnhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}

func (s Status) GoString() string {
	return "healthcheck.Status(" + strconv.Itoa(int(s)) + ")"
}

// Combine [Status] a and b and determine the combined status of both on below
// rules:
//   - when a or b is [StatusUnhealthy], the result is [StatusUnhealthy];
//   - when a is [StatusHealthy] and b is not, the result is [StatusUnhealthy];
//   - when a is [StatusUnknown], the result is b;
//   - when b is [StatusHealthy], the result is [StatusHealthy];
//   - all other cases result in [StatusUnknown]
func Combine(a, b Status) Status {
	if a == StatusUnhealthy || b == StatusUnhealthy || (a == StatusHealthy && b != StatusHealthy) {
		return StatusUnhealthy
	}
	if a == StatusUnknown || b == StatusHealthy {
		return b
	}

	return StatusUnknown
}

// AtomicStatus is an atomic [Status]. The zero value is [StatusUnknown].
type AtomicStatus struct {
	x atomic.Int32
}

func (x *AtomicStatus) Combine(stat Status) Status {
	stat = Combine(x.Load(), stat)
	x.Store(stat)
	return stat
}

// Load atomically loads and returns the [Status] stored in x.
func (x *AtomicStatus) Load() Status { return Status(x.x.Load()) }

// Store atomically stores val into x.
func (x *AtomicStatus) Store(val Status) { x.x.Store(int32(val)) }

// Swap atomically stores new into x and returns the previous [Status].
func (x *AtomicStatus) Swap(new Status) (old Status) {
	return Status(x.x.Swap(int32(new)))
}

// CompareAndSwap executes the compare-and-swap operation for x.
func (x *AtomicStatus) CompareAndSwap(old, new Status) (swapped bool) {
	return x.x.CompareAndSwap(int32(old), int32(new))
}

func (x *AtomicStatus) GoString() string {
	return "healthcheck.AtomicStatus(" + strconv.Itoa(int(x.Load())) + ")"
}
