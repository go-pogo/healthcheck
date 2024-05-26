// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthclient

import (
	"time"
)

type Config struct {
	// TargetHostname is the hostname or ip address of the target server.
	TargetHostname string `env:"" default:"localhost"`
	// TargetPort is the port of the target server.
	TargetPort uint16 `env:"" default:"8080"`
	// TargetPath is the path to the health check endpoint on the target server.
	TargetPath string `env:"" default:"/healthy"`
	// RequestTimeout is the maximum time to wait for a health check response.
	RequestTimeout time.Duration `env:"" default:"3s"`
}

var defaultConfig = Config{
	TargetHostname: "localhost",
	TargetPort:     80,
	TargetPath:     "/healthy",
	RequestTimeout: 3 * time.Second,
}

func DefaultConfig() Config { return defaultConfig }
