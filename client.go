// Copyright (c) 2022, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

import (
	"context"
	"crypto/tls"
	"github.com/go-pogo/errors"
	"github.com/go-pogo/serv"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type InvalidStatusCode struct {
	Code int
}

func (e InvalidStatusCode) Error() string {
	return "invalid status code " + strconv.Itoa(e.Code)
}

type ClientConfig struct {
	Host    string `default:"localhost"`
	Port    serv.Port
	Path    string        `default:"/healthy"`
	Timeout time.Duration `default:"3s"`
}

type Client struct {
	ClientConfig

	httpClient *http.Client
	tls        bool
}

func NewClient(tlsConf *tls.Config) *Client {
	c := &Client{
		httpClient: &http.Client{},
	}
	if tlsConf != nil {
		c.httpClient.Transport = &http.Transport{TLSClientConfig: tlsConf}
		c.tls = true
	}
	return c
}

func (c *Client) newRequest() *http.Request {
	req := http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "http",
			Host:   c.Host,
			Path:   c.Path,
		},
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Host:       c.Host,
	}
	if c.tls {
		req.URL.Scheme = "https"
	}
	if c.Port != 0 {
		req.URL.Host = serv.JoinHostPort(req.URL.Host, c.Port)
	}
	return &req
}

func (c *Client) Request(ctx context.Context) (Status, error) {
	timeout := c.ClientConfig.Timeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}
	if t, ok := ctx.Deadline(); !ok || timeout < time.Until(t) {
		var cancelFn context.CancelFunc
		ctx, cancelFn = context.WithTimeout(ctx, timeout)
		defer cancelFn()
	}

	resp, err := c.httpClient.Do(c.newRequest().WithContext(ctx))
	if err != nil {
		return StatusUnknown, errors.WithStack(err)
	}

	_ = resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusTooEarly:
		return StatusUnknown, nil
	case http.StatusOK, http.StatusNoContent:
		return StatusHealthy, nil
	case http.StatusServiceUnavailable:
		return StatusUnhealthy, nil
	default:
		return StatusUnknown, errors.WithStack(&InvalidStatusCode{
			Code: resp.StatusCode,
		})
	}
}
