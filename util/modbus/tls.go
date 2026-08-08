package modbus

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"

	gridx "github.com/grid-x/modbus"
	"github.com/volkszaehler/mbmd/meters"
)

// tlsConfig builds a *tls.Config for Modbus over mutual TLS (mTLS) from the
// PEM-encoded certificates in Settings.
//
// ClientCert and ClientKey are required since Modbus/TLS mandates client
// authentication. CACert verifies the device certificate. Devices typically
// present a self-signed certificate, for which Insecure must be set instead.
func (s Settings) tlsConfig() (*tls.Config, error) {
	if s.ClientCert == "" || s.ClientKey == "" {
		return nil, errors.New("modbus tls: clientcert and clientkey required")
	}

	if s.CACert == "" && !s.Insecure {
		return nil, errors.New("modbus tls: cacert required to verify the device certificate, use insecure to skip verification")
	}

	clientCert, err := tls.X509KeyPair([]byte(s.ClientCert), []byte(s.ClientKey))
	if err != nil {
		return nil, fmt.Errorf("modbus tls: client certificate: %w", err)
	}

	cfg := &tls.Config{
		Certificates:       []tls.Certificate{clientCert},
		InsecureSkipVerify: s.Insecure,
		MinVersion:         tls.VersionTLS12,
	}

	if s.CACert != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(s.CACert)) {
			return nil, errors.New("modbus tls: no certificate found in cacert")
		}
		cfg.RootCAs = pool
	}

	return cfg, nil
}

// newTLSConn creates a Modbus TCP connection wrapped in TLS
func newTLSConn(uri string, tlsConfig *tls.Config) *meters.TCP {
	handler := gridx.NewTCPClientHandler(uri, gridx.WithTLSConfig(tlsConfig))

	// use retry outside of grid-x/modbus
	handler.LinkRecoveryTimeout = 0
	handler.ProtocolRecoveryTimeout = 0

	return &meters.TCP{
		Client:  gridx.NewClient(handler),
		Handler: handler,
	}
}
