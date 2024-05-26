// Copyright (c) 2022, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthclient

import (
	"context"
	"crypto/tls"
	"github.com/go-pogo/errors"
	"github.com/go-pogo/healthcheck"
	"net"
	"net/http"
	"strconv"
	"strings"
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

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
	client() *http.Client
	transport() *http.Transport
}

// Client is a simple http.Client which can be used to perform health checks
// on a target (web)service.
type Client struct {
	Config

	httpClient        httpClient
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
	if c.httpClient == nil {
		return nil
	}
	if t := c.httpClient.transport(); t != nil {
		return t.TLSClientConfig
	}
	return nil
}

func (c *Client) newRequest(ctx context.Context) (*http.Request, error) {
	var host string
	if c.bindTargetBaseURL != nil {
		host = *c.bindTargetBaseURL
	} else {
		host = c.Config.TargetHostname
		if host == "" {
			host = "localhost"
		}
		if c.TargetPort != 0 {
			host = net.JoinHostPort(host, strconv.FormatUint(uint64(c.TargetPort), 10))
		}
	}
	if !strings.Contains(host, "://") {
		//goland:noinspection HttpUrlsUsage
		host = "http://" + host
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, host, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if c.TLSConfig() != nil {
		req.URL.Scheme = "https"
	}

	req.URL.Path = unbind(c.TargetPath, c.bindTargetPath)
	return req, nil
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

	req, err := c.newRequest(ctx)
	if err != nil {
		return healthcheck.StatusUnknown, errors.WithStack(err)
	}

	if c.httpClient == nil {
		c.httpClient = &wrappedHTTPClient{http.DefaultClient}
	}

	resp, err := c.httpClient.Do(req)
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

var _ httpClient = (*wrappedHTTPClient)(nil)

type wrappedHTTPClient struct {
	*http.Client
}

func (c *wrappedHTTPClient) client() *http.Client { return c.Client }

func (c *wrappedHTTPClient) transport() *http.Transport {
	if c.Transport == nil {
		return nil
	}
	if t, ok := c.Transport.(*http.Transport); ok {
		return t
	}
	return nil
}
