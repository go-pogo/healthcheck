// Copyright (c) 2022, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

import (
	"github.com/go-pogo/errors"
	"github.com/go-pogo/serv"
	"net/http"
	"time"
)

const (
	// Route is the default path for a http handler.
	Route = "/health"

	Header = "X-Service-Health"
)

type Listener interface {
	HealthChanged(status, oldStatus Status)
}

type Option func(c *Config) error

type Config struct {
	HttpClient    *http.Client
	Listener      Listener
	Flag          string
	Host          string
	Port          serv.Port
	Path          string
	Header        string
	Timeout       time.Duration
	CheckParallel bool
}

func New(opts ...Option) (*Config, error) {
	var c Config
	return &c, c.applyOptions(opts)
}

func WithHttpClient(client *http.Client) Option {
	return func(c *Config) error {
		c.HttpClient = client
		return nil
	}
}

func WithListener(l Listener) Option {
	return func(c *Config) error {
		c.Listener = l
		return nil
	}
}

func WithHost(host string) Option {
	return func(c *Config) error {
		c.Host = host
		return nil
	}
}

func WithPort(port serv.Port) Option {
	return func(c *Config) error {
		c.Port = port
		return nil
	}
}

func WithHostPort(hostport string) Option {
	return func(c *Config) error {
		h, p, err := serv.SplitHostPort(hostport)
		if err != nil {
			return err
		}
		c.Host = h
		c.Port = p
		return nil
	}
}

func WithPath(path string) Option {
	return func(c *Config) error {
		c.Path = path
		return nil
	}
}

func WithHeader(header string) Option {
	return func(c *Config) error {
		c.Header = header
		return nil
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) error {
		c.Timeout = timeout
		return nil
	}
}

func (c *Config) applyOptions(opts []Option) error {
	var err error
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		errors.Append(&err, opt(c))
	}
	if err == nil {
		c.defaults()
	}

	return err
}

func (c *Config) defaults() {
	if c.HttpClient == nil {
		c.HttpClient = http.DefaultClient
	}
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Path == "" {
		c.Path = Route
	}
	if c.Header == "" {
		c.Header = Header
	}
	if c.Timeout == 0 {
		c.Timeout = time.Second * 3
	}
}
