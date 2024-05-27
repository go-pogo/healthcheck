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

const ErrUnknownTransportType errors.Msg = "cannot add tls.Config to http.Client.Transport of unknown type"

type Option func(c *Client) error

// WithHTTPClient allows to set a custom internal http.Client to the [Client].
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) error {
		c.httpClient = httpClient
		return nil
	}
}

const panicNilTLSConfig = "healthcheck.WithTLSConfig: tls.Config should not be nil"

// WithTLSConfig sets the provided [tls.Config] to the [Client]'s internal
// [http.Transport.TLSClientConfig]. Any provided [easytls.Option](s) will be
// applied to this [tls.Config] using [easytls.Apply].
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
			return errors.New(ErrUnknownTransportType)
		}

		return easytls.Apply(conf, easytls.TargetClient, opts...)
	}
}

// WithBindTargetBaseURL where ptr points to a strings which contains the base
// url to the target server, of form "[scheme://]ipaddr|hostname[:port]",
// without trailing slash.
func WithBindTargetBaseURL(ptr *string) Option {
	return func(c *Client) error {
		c.bindTargetBaseURL = ptr
		return nil
	}
}

func WithBindTargetPath(ptr *string) Option {
	return func(c *Client) error {
		c.bindTargetPath = ptr
		return nil
	}
}
