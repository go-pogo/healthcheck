// Copyright (c) 2022, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthclient

import (
	"context"
	"crypto/tls"
	"github.com/go-pogo/errors"
	"github.com/go-pogo/healthcheck"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"
)

type InvalidStatusCode struct {
	Code int
}

func (e InvalidStatusCode) Error() string {
	return "invalid status code " + strconv.Itoa(e.Code)
}

type Config struct {
	// BaseURL of form "[scheme://]ipaddr[:port]" or
	// "[scheme://]hostname[:port]", both without trailing slash.
	BaseURL string        `env:"" default:"localhost"`
	Path    string        `env:"" default:"/healthy"`
	Timeout time.Duration `env:"" default:"3s"`
}

type Client struct {
	Config

	httpClient  *http.Client
	bindBaseURL *string
}

func New(conf Config, opts ...Option) (*Client, error) {
	c := Client{Config: conf}
	return &c, c.With(opts...)
}

func (c *Client) With(opts ...Option) error {
	var err error
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		err = errors.Append(err, opt(c))
	}
	return err
}

func (c *Client) TLSConfig() *tls.Config {
	if c.httpClient == nil || c.httpClient.Transport == nil {
		return nil
	}
	if t, ok := c.httpClient.Transport.(*http.Transport); ok {
		return t.TLSClientConfig
	}
	return nil
}

func (c *Client) newRequest() (*http.Request, error) {
	base := c.BaseURL
	if c.bindBaseURL != nil {
		base = *c.bindBaseURL
	}

	u, err := url.ParseRequestURI(base)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if c.TLSConfig() != nil {
		u.Scheme = "https"
	} else if u.Scheme == "" {
		u.Scheme = "http"
	}
	if u.Host == "" {
		u.Host = "localhost"
	}
	if u.Path != "" {
		u.Path = path.Join(u.Path, c.Path)
	}

	return &http.Request{
		Method:     http.MethodGet,
		URL:        u,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Host:       u.Host,
	}, nil
}

func (c *Client) Request(ctx context.Context) (healthcheck.Status, error) {
	timeout := c.Config.Timeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}
	if t, ok := ctx.Deadline(); !ok || timeout < time.Until(t) {
		var cancelFn context.CancelFunc
		ctx, cancelFn = context.WithTimeout(ctx, timeout)
		defer cancelFn()
	}

	req, err := c.newRequest()
	if err != nil {
		return healthcheck.StatusUnknown, errors.WithStack(err)
	}

	httpClient := c.httpClient
	if c.httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return healthcheck.StatusUnknown, errors.WithStack(err)
	}

	_ = resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusTooEarly:
		return healthcheck.StatusUnknown, nil
	case http.StatusOK, http.StatusNoContent:
		return healthcheck.StatusHealthy, nil
	case http.StatusServiceUnavailable:
		return healthcheck.StatusUnhealthy, nil
	default:
		return healthcheck.StatusUnknown, errors.WithStack(&InvalidStatusCode{
			Code: resp.StatusCode,
		})
	}
}
