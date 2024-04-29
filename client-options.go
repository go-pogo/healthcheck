// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/go-pogo/errors"
	"net/http"
)

type ClientOption interface {
	apply(c *Client) error
}

type clientOptionFunc func(c *Client) error

func (fn clientOptionFunc) apply(c *Client) error { return fn(c) }

func WithHTTPClient(httpClient *http.Client) ClientOption {
	return clientOptionFunc(func(c *Client) error {
		c.httpClient = httpClient
		return nil
	})
}

func WithTLSConfig(tlsConf *tls.Config) ClientOption {
	return clientOptionFunc(func(c *Client) error {
		if c.httpClient == nil {
			c.httpClient = &http.Client{
				Transport: &http.Transport{TLSClientConfig: tlsConf},
			}
			return nil
		}
		if c.httpClient.Transport == nil {
			c.httpClient.Transport = &http.Transport{TLSClientConfig: tlsConf}
			return nil
		}
		if t, ok := c.httpClient.Transport.(*http.Transport); ok {
			t.TLSClientConfig = tlsConf
			return nil
		}

		return errors.New("cannot add tls.Config to http.Client.Transport of unknown type")
	})
}

func WithTLSRootCAs(certs ...tls.Certificate) ClientOption {
	return clientOptionFunc(func(c *Client) error {
		tlsConf := c.tlsConfig()
		if tlsConf == nil {
			tlsConf = new(tls.Config)
			if err := WithTLSConfig(tlsConf).apply(c); err != nil {
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
	})
}

func WithBindBaseURL(ptr *string) ClientOption {
	return clientOptionFunc(func(c *Client) error {
		c.bindBaseURL = ptr
		return nil
	})
}
