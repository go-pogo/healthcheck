// Copyright (c) 2022, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthclient

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	urlpkg "net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-pogo/errors"
	"github.com/go-pogo/healthcheck"
)

const (
	ErrCreateClient   errors.Msg = "failed to create healthcheck client"
	ErrApplyOptions   errors.Msg = "error while applying option(s)"
	ErrInvalidBaseURL errors.Msg = "invalid bound base url"
	ErrRequestFailed  errors.Msg = "request failed"
)

// InvalidStatusCode contains the non-expected status code received from a
// health check request.
type InvalidStatusCode struct {
	Code int
}

func (e InvalidStatusCode) Error() string {
	return "invalid status code " + strconv.Itoa(e.Code)
}

// Request creates a new [Client] and performs a request based on the provided
// [Config] and [Option(s)].
func Request(ctx context.Context, conf Config, opts ...Option) (healthcheck.Status, error) {
	client, err := New(conf, opts...)
	if err != nil {
		return healthcheck.StatusUnknown, errors.Wrap(err, ErrCreateClient)
	}
	return client.Request(ctx)
}

// Client is a wrapper around [http.Client] which can be used to perform health
// checks on a target (web)service.
type Client struct {
	Config

	log               Logger
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
	return errors.Wrap(err, ErrApplyOptions)
}

func (c *Client) TargetURL() (*urlpkg.URL, error) {
	var url *urlpkg.URL
	var err error

	if c.bindTargetBaseURL != nil {
		baseURL := *c.bindTargetBaseURL
		if !strings.Contains(baseURL, "://") {
			//goland:noinspection HttpUrlsUsage
			baseURL = "http://" + baseURL
		}

		url, err = urlpkg.Parse(baseURL)
		if err != nil {
			return nil, errors.Wrap(err, ErrInvalidBaseURL)
		}
	} else {
		url = &urlpkg.URL{
			Scheme: "http",
			Host:   c.Config.TargetHostname,
		}
		if url.Host == "" {
			url.Host = "localhost"
		}
		if c.TargetPort != 0 {
			url.Host = net.JoinHostPort(url.Host, strconv.FormatUint(uint64(c.TargetPort), 10))
		}
	}

	if c.TLSConfig() != nil {
		url.Scheme = "https"
	}
	if c.bindTargetPath != nil {
		url.Path = *c.bindTargetPath
	} else {
		url.Path = c.TargetPath
	}

	return url, nil
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

// Request performs a [http.Request] and determines the service's
// [healthcheck.Status] on the received http status codes. The status is
// [healthcheck.StatusHealthy] when the http status is either [http.StatusOK]
// or [http.StatusNoContent]. The status is [healthcheck.StatusUnhealthy] when
// the http status is [http.StatusServiceUnavailable].
// Any error during the request, or any other received http status, results in
// a status of [healthcheck.StatusUnknown].
func (c *Client) Request(ctx context.Context) (stat healthcheck.Status, err error) {
	if c.log != nil {
		defer func() {
			// todo: test if this works
			if err != nil {
				c.log.LogHealthCheckFailed(stat, err)
			} else {
				c.log.LogHealthChecked(stat)
			}
		}()
	}

	timeout := c.Config.RequestTimeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}
	if t, ok := ctx.Deadline(); !ok || timeout < time.Until(t) {
		var cancelFn context.CancelFunc
		ctx, cancelFn = context.WithTimeout(ctx, timeout)
		defer cancelFn()
	}

	url, err := c.TargetURL()
	if err != nil {
		return healthcheck.StatusUnknown, errors.Wrap(err, ErrRequestFailed)
	}

	req := &http.Request{
		Method:     http.MethodGet,
		URL:        url,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Host:       url.Host,
	}

	if c.httpClient == nil {
		c.httpClient = http.DefaultClient
	}

	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return healthcheck.StatusUnknown, errors.Wrap(err, ErrRequestFailed)
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
