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

// InvalidStatusCode contains the non-expected status code received from a
// health check request.
type InvalidStatusCode struct {
	Code int
}

func (e InvalidStatusCode) Error() string {
	return "invalid status code " + strconv.Itoa(e.Code)
}

// Client is a simple http.Client which can be used to perform health checks
// on a target (web)service.
type Client struct {
	Config

	httpClient        *http.Client
	bindTargetBaseURL *string
	bindTargetPath    *string
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
	u, err := url.ParseRequestURI(unbind(c.TargetBaseURL, c.bindTargetBaseURL))
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

	if targetPath := unbind(c.TargetPath, c.bindTargetPath); u.Path != "" {
		u.Path = path.Join(u.Path, targetPath)
	} else {
		u.Path = targetPath
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
	timeout := c.Config.RequestTimeout
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

func unbind(def string, ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return def
}
