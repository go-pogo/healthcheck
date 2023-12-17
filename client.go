// Copyright (c) 2022, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

import (
	"context"
	"github.com/go-pogo/errors"
	"github.com/go-pogo/serv"
	"net/http"
	"path"
	"strings"
	"time"
)

type Client struct {
	c *Config
}

func (c *Config) Client() *Client { return &Client{c: c} }

func (c *Client) Request(ctx context.Context) (Status, error) {
	c.c.defaults()
	ctx, cancelFn := timeoutContext(ctx, c.c.Timeout)
	if cancelFn != nil {
		defer cancelFn()
	}

	var addr string
	if c.c.Port == 0 {
		addr = c.c.Host
	} else {
		addr = serv.JoinHostPort(c.c.Host, c.c.Port)
	}

	addr = path.Join(addr, c.c.Path)
	if !strings.HasPrefix(addr, "http") {
		addr = "http://" + addr
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr, nil)
	if err != nil {
		return Error, errors.WithStack(err)
	}

	resp, err := c.c.HttpClient.Do(req)
	if err != nil {
		return Error, errors.WithStack(err)
	}

	_ = resp.Body.Close()

	var state Status
	if header := resp.Header.Get(c.c.Header); header != "" {
		state = ParseStatus(header)
	} else {
		state = StatusCode(resp.StatusCode)
	}
	return state, nil
}

func timeoutContext(ctx context.Context, t time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if t != 0 {
		if dl, ok := ctx.Deadline(); !ok || t < time.Until(dl) {
			return context.WithTimeout(ctx, t)
		}
	}
	return ctx, nil
}
