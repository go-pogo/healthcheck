// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthclient

import (
	"crypto/tls"
	"github.com/go-pogo/easytls"
	"github.com/go-pogo/errors"
	"net/http"
)

type Option func(c *Client) error

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) error {
		c.httpClient = httpClient
		return nil
	}
}

const panicNilTLSConfig = "healthcheck.WithTLSConfig: tls.Config should not be nil"

// WithTLSConfig sets the provided [tls.Config] to the [Client]'s internal
// [http.Transport.TLSClientConfig]. Any provided [TLSOption](s) will be
// applied to this [tls.Config].
func WithTLSConfig(conf *tls.Config, opts ...easytls.Option) Option {
	return func(c *Client) error {
		if conf == nil {
			panic(panicNilTLSConfig)
		}

		if c.httpClient == nil {
			c.httpClient = &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: conf,
				},
			}
		} else if c.httpClient.Transport == nil {
			c.httpClient.Transport = &http.Transport{
				TLSClientConfig: conf,
			}
		} else if t, ok := c.httpClient.Transport.(*http.Transport); ok {
			t.TLSClientConfig = conf
		} else {
			return errors.New("cannot add tls.Config to http.Client.Transport of unknown type")
		}

		return easytls.Apply(conf, easytls.TargetClient, opts...)
	}
}

func WithBindBaseURL(ptr *string) Option {
	return func(c *Client) error {
		c.bindBaseURL = ptr
		return nil
	}
}
