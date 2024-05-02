// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthclient

import (
	"crypto/tls"
	"crypto/x509"
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

func WithTLSConfig(conf *tls.Config) Option {
	return func(c *Client) error {
		if conf == nil {
			panic(panicNilTLSConfig)
		}

		if c.httpClient == nil {
			c.httpClient = &http.Client{
				Transport: &http.Transport{TLSClientConfig: conf},
			}
			return nil
		}
		if c.httpClient.Transport == nil {
			c.httpClient.Transport = &http.Transport{TLSClientConfig: conf}
			return nil
		}
		if t, ok := c.httpClient.Transport.(*http.Transport); ok {
			t.TLSClientConfig = conf
			return nil
		}

		return errors.New("cannot add tls.Config to http.Client.Transport of unknown type")
	}
}

func WithTLSRootCAs(certs ...tls.Certificate) Option {
	return func(c *Client) error {
		if len(certs) == 0 {
			return nil
		}

		tlsConf := c.TLSConfig()
		if tlsConf == nil {
			tlsConf = new(tls.Config)
			if err := WithTLSConfig(tlsConf)(c); err != nil {
				return err
			}
		}
		if tlsConf.RootCAs == nil {
			tlsConf.RootCAs = x509.NewCertPool()
		}

		for _, cert := range certs {
			x, err := x509.ParseCertificate(cert.Certificate[0])
			if err != nil {
				return err
			}
			tlsConf.RootCAs.AddCert(x)
		}
		return nil
	}
}

func WithBindBaseURL(ptr *string) Option {
	return func(c *Client) error {
		c.bindBaseURL = ptr
		return nil
	}
}
