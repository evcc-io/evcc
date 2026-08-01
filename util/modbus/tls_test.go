package modbus

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// testKeyPair creates a self-signed certificate and key in PEM format
func testKeyPair(t *testing.T) (string, string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "evcc"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IsCA:         true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	keyDer, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDer})

	return string(certPem), string(keyPem)
}

func TestSettingsTlsConfigured(t *testing.T) {
	tc := []struct {
		Settings
		res bool
	}{
		{Settings{}, false},
		{Settings{URI: "foo"}, false},
		{Settings{ClientCert: "foo"}, true},
		{Settings{ClientKey: "foo"}, true},
		{Settings{CACert: "foo"}, true},
		{Settings{Insecure: true}, true},
	}

	for _, tc := range tc {
		require.Equal(t, tc.res, tc.tlsConfigured(), tc)
	}
}

func TestTlsConfig(t *testing.T) {
	cert, key := testKeyPair(t)

	t.Run("insecure", func(t *testing.T) {
		cfg, err := Settings{ClientCert: cert, ClientKey: key, Insecure: true}.tlsConfig()
		require.NoError(t, err)
		require.True(t, cfg.InsecureSkipVerify)
		require.Nil(t, cfg.RootCAs)
		require.Len(t, cfg.Certificates, 1)
	})

	t.Run("cacert", func(t *testing.T) {
		cfg, err := Settings{ClientCert: cert, ClientKey: key, CACert: cert}.tlsConfig()
		require.NoError(t, err)
		require.False(t, cfg.InsecureSkipVerify)
		require.NotNil(t, cfg.RootCAs)
	})

	// incomplete or invalid configuration must error instead of silently
	// falling back to an unencrypted connection
	t.Run("errors", func(t *testing.T) {
		tc := []struct {
			name string
			Settings
		}{
			{"cert without key", Settings{ClientCert: cert, Insecure: true}},
			{"key without cert", Settings{ClientKey: key, Insecure: true}},
			{"cacert only", Settings{CACert: cert}},
			{"insecure only", Settings{Insecure: true}},
			{"neither cacert nor insecure", Settings{ClientCert: cert, ClientKey: key}},
			{"invalid keypair", Settings{ClientCert: "foo", ClientKey: "bar", Insecure: true}},
			{"invalid cacert", Settings{ClientCert: cert, ClientKey: key, CACert: "foo"}},
		}

		for _, tc := range tc {
			t.Run(tc.name, func(t *testing.T) {
				_, err := tc.tlsConfig()
				require.Error(t, err)
			})
		}
	})
}

// TestTlsRequiresTcp ensures tls settings are rejected for non-tcp transports
// instead of being silently ignored
func TestTlsRequiresTcp(t *testing.T) {
	cert, key := testKeyPair(t)
	tls := Settings{ClientCert: cert, ClientKey: key, Insecure: true}

	tc := []struct {
		name string
		Settings
	}{
		{"serial device", Settings{Device: "/dev/ttyUSB0", Comset: "8N1", Baudrate: 9600}},
		{"rtu over tcp", Settings{URI: "foo:502", RTU: new(true)}},
		{"udp", Settings{URI: "foo:502", UDP: true}},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tc.Settings
			cfg.ClientCert, cfg.ClientKey, cfg.Insecure = tls.ClientCert, tls.ClientKey, tls.Insecure

			_, err := physicalConnection(context.Background(), cfg.Protocol(), cfg)
			require.ErrorContains(t, err, "tls requires tcp")
		})
	}
}
