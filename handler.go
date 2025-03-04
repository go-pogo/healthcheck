// Copyright (c) 2023, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

import (
	"encoding/json"
	"net/http"
)

// PathPattern is the default path for a http handler.
const PathPattern = "/healthy"

var okBytes = []byte("ok")

// SimpleHTTPHandler is a [http.Handler] that writes a default "ok" message.
func SimpleHTTPHandler() http.Handler {
	return http.HandlerFunc(func(wri http.ResponseWriter, _ *http.Request) {
		_, _ = wri.Write(okBytes)
	})
}

// HTTPHandler returns a [http.Handler] that writes the health status of the
// provided [HealthChecker] hc. If hc is a [Checker], the response will be a
// json object containing the individual statuses of all registered
// [HealthChecker](s) in hc when health status is not [StatusHealthy].
func HTTPHandler(hc HealthChecker) http.Handler {
	if hc == nil {
		panic(panicNilHealthChecker)
	}

	if checker, ok := hc.(*Checker); ok {
		return http.HandlerFunc(func(wri http.ResponseWriter, req *http.Request) {
			stat := checker.CheckHealth(req.Context())
			if stat == StatusHealthy {
				wri.WriteHeader(stat.StatusCode())
				_, _ = wri.Write(okBytes)
			} else {
				wri.Header().Set("Content-Type", "application/json")
				wri.WriteHeader(stat.StatusCode())
				_ = json.NewEncoder(wri).Encode(checker.Details())
			}
		})
	}

	return http.HandlerFunc(func(wri http.ResponseWriter, req *http.Request) {
		stat := hc.CheckHealth(req.Context())
		wri.WriteHeader(stat.StatusCode())

		if stat == StatusHealthy {
			_, _ = wri.Write(okBytes)
		} else {
			_, _ = wri.Write([]byte(stat.String()))
		}
	})
}
