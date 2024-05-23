// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthclient

import "time"

type Config struct {
	// TargetBaseURL is the base url to the target server, of form
	// "[scheme://]ipaddr|hostname[:port]", without trailing slash.
	TargetBaseURL string `env:"" default:"localhost"`
	// TargetPath is the path to the health check endpoint on the target server.
	TargetPath string `env:"" default:"/healthy"`
	// RequestTimeout is the maximum time to wait for a health check response.
	RequestTimeout time.Duration `env:"" default:"3s"`
}

var defaultConfig = Config{
	TargetBaseURL:  "localhost",
	TargetPath:     "/healthy",
	RequestTimeout: 3 * time.Second,
}

func DefaultConfig() Config { return defaultConfig }
