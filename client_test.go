// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
)

func TestClient_Request(t *testing.T) {
	t.Run("without tls", func(t *testing.T) {
		srv := httptest.NewServer(SimpleHTTPHandler())
		defer srv.Close()

		client, err := NewClient(ClientConfig{BaseURL: srv.URL})
		assert.NoError(t, err)

		stat, err := client.Request(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, StatusHealthy, stat)
	})
	t.Run("with tls", func(t *testing.T) {
		srv := httptest.NewTLSServer(SimpleHTTPHandler())
		defer srv.Close()

		client, err := NewClient(ClientConfig{},
			WithBindBaseURL(&srv.URL),
			WithTLSConfig(&tls.Config{
				RootCAs:    x509.NewCertPool(),
				ClientAuth: tls.RequireAndVerifyClientCert, // mTLS
			}),
			WithTLSRootCAs(srv.TLS.Certificates[0]),
		)
		assert.NoError(t, err)

		stat, err := client.Request(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, StatusHealthy, stat)
	})
}
