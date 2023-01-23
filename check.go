// Copyright (c) 2022, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

import (
	"context"
	"flag"
	"fmt"
	"os"
)

//goland:noinspection GoUnusedConst
const (
	// ShortFlag is the default flag to print the current build information of the app.
	ShortFlag = "H"
	// LongFlag is an alternative long version that may be used together with ShortFlag.
	LongFlag = "health"
)

// Check creates a new Config using the provided options, and requests the state
// of the running app when a ShortFlag or LongFlag is provided as os.Args. It
// then exits with a Status as code.
func Check(ctx context.Context, opts ...Option) (*Config, error) {
	c, err := New(opts...)
	if err != nil {
		return nil, err
	}

	flags := flag.NewFlagSet("", flag.ContinueOnError)

	var do bool
	if c.Flag == "" {
		flags.BoolVar(&do, ShortFlag, false, "")
		flags.BoolVar(&do, LongFlag, false, "")
	} else {
		flags.BoolVar(&do, c.Flag, false, "")
	}

	_ = flags.Parse(os.Args[1:])
	if do {
		c.Client().Check(ctx)
	}
	return c, nil
}

// Check requests the state of the running app and exits with a Status as code.
func (c *Client) Check(ctx context.Context) {
	state, err := c.Request(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%+v", err)
	}
	state.Exit()
}
