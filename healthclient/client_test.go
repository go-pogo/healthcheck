// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/go-pogo/easytls"
	"github.com/go-pogo/healthcheck"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestClient_TargetURL(t *testing.T) {
	tests := map[string]struct {
		conf    Config
		opts    []Option
		wantURL url.URL
		wantErr error
	}{
		"target hostname": {
			conf: Config{TargetHostname: "testerdetest"},
			wantURL: url.URL{
				Scheme: "http",
				Host:   "testerdetest",
			},
		},
		"target port": {
			conf: Config{TargetPort: 1234},
			wantURL: url.URL{
				Scheme: "http",
				Host:   "localhost:1234",
			},
		},
		"target port with tls": {
			conf: Config{TargetPort: 1234},
			opts: []Option{WithTLSConfig(&tls.Config{})},
			wantURL: url.URL{
				Scheme: "https",
				Host:   "localhost:1234",
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			client, err := New(test.conf, test.opts...)
			assert.NoError(t, err)

			haveURL, haveErr := client.TargetURL()
			assert.Equal(t, test.wantURL, *haveURL)
			if test.wantErr == nil {
				assert.NoError(t, haveErr)
			} else {
				assert.ErrorIs(t, haveErr, test.wantErr)
			}
		})
	}
}

func TestClient_Request(t *testing.T) {
	t.Run("without tls", func(t *testing.T) {
		srv := httptest.NewServer(healthcheck.SimpleHTTPHandler())
		defer srv.Close()

		client, err := New(Config{}, WithBindTargetBaseURL(&srv.URL))
		assert.NoError(t, err)

		stat, err := client.Request(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, healthcheck.StatusHealthy, stat)
	})
	t.Run("with tls", func(t *testing.T) {
		srv := httptest.NewTLSServer(healthcheck.SimpleHTTPHandler())
		defer srv.Close()

		client, err := New(Config{},
			WithBindTargetBaseURL(&srv.URL),
			WithTLSConfig(
				&tls.Config{
					RootCAs:    x509.NewCertPool(),
					ClientAuth: tls.RequireAndVerifyClientCert, // mTLS
				},
				easytls.WithTLSRootCAs(srv.TLS.Certificates[0]),
			),
		)
		assert.NoError(t, err)

		stat, err := client.Request(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, healthcheck.StatusHealthy, stat)
	})
}
