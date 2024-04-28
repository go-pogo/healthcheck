// Copyright (c) 2023, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

import (
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

type handler struct {
	check StatusChecker
}

const panicNilStatusChecker = "healthcheck.HTTPHandler: StatusChecker should not be nil"

func HTTPHandler(c StatusChecker) http.Handler {
	if c == nil {
		panic(panicNilStatusChecker)
	}
	return &handler{check: c}
}

func (h *handler) ServeHTTP(wri http.ResponseWriter, req *http.Request) {
	stat := h.check.CheckStatus(req.Context())
	wri.WriteHeader(stat.StatusCode())
	_, _ = wri.Write([]byte(stat.String()))
}
