// Copyright (c) 2022, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

import (
	"net/http"
	"os"
	"strconv"
)

// Status indicates the health status of the container.
// https://docs.docker.com/engine/reference/builder/#healthcheck
type Status int

const (
	Unknown Status = -1
	// Healthy indicates the container is healthy and ready for use.
	Healthy Status = 0
	// Unhealthy indicates the container is not working correctly.
	Unhealthy Status = 1
	// Error indicates an error occurred while performing the health check.
	// This result should be treated as a panic.
	Error Status = 3

	ok        = "ok"
	unhealthy = "unhealthy"
	unknown   = "unknown"
)

func ParseStatus(v string) Status {
	switch v {
	case ok:
		return Healthy
	case unhealthy:
		return Unhealthy
	case "", unknown:
		return Unknown
	}
	return Error
}

// StatusCode returns a Status representing the given http status code.
func StatusCode(c int) Status {
	if c == http.StatusInternalServerError {
		return Error
	}
	if c < 400 {
		return Healthy
	}
	return Unhealthy
}

// StatusCode returns a http statuscode which represents Status.
func (r Status) StatusCode() int {
	switch r {
	case Healthy:
		return http.StatusOK // 200
	case Unhealthy:
		return http.StatusServiceUnavailable // 503
	case Unknown:
		return http.StatusTooEarly // 425
	default:
		return http.StatusInternalServerError // 500
	}
}

// String return a string representation of Status.
func (r Status) String() string {
	switch r {
	case Healthy:
		return ok
	case Unhealthy:
		return unhealthy
	case Unknown:
		return unknown
	default:
		return "error"
	}
}

func (r Status) GoString() string {
	return "healthcheck.Status(" + strconv.Itoa(int(r)) + ")"
}

// Exit causes the current program to exit with Status as the given status code.
// The program terminates immediately; deferred functions are not run.
func (r Status) Exit() { os.Exit(int(r)) }
