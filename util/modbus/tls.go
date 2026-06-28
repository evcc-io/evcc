package modbus

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/evcc-io/evcc/util"
	gridx "github.com/grid-x/modbus"
	"github.com/volkszaehler/mbmd/meters"
)

// TLSConfig builds a *tls.Config for Modbus over mutual TLS (mTLS).
//
// cert and key are required: they provide the client certificate used for
// mTLS.
//
// ca is optional. When non-empty it is loaded as the trusted CA for server
// certificate verification. When empty the server certificate is not verified
// (InsecureSkipVerify), which is typical for devices that present a private-CA
// or self-signed server certificate not in the system trust store.
func TLSConfig(cert, key, ca string) (*tls.Config, error) {
	if cert == "" || key == "" {
		return nil, fmt.Errorf("modbus tls: client certificate (cert and key) required for mTLS")
	}

	clientCert, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return nil, fmt.Errorf("modbus tls: loading client certificate: %w", err)
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		MinVersion:   tls.VersionTLS12,
	}

	if ca == "" {
		cfg.InsecureSkipVerify = true
	} else {
		pem, err := os.ReadFile(ca)
		if err != nil {
			return nil, fmt.Errorf("modbus tls: reading ca certificate: %w", err)
		}

		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("modbus tls: no certificates found in %s", ca)
		}
		cfg.RootCAs = pool
	}

	return cfg, nil
}

// NewConnectionFromSettings creates a modbus connection from settings, using
// TLS when a client certificate is configured (Settings.TLS) and the plain
// transport otherwise.
func NewConnectionFromSettings(ctx context.Context, cfg Settings) (*Connection, error) {
	if cfg.TLS() {
		tlsConfig, err := TLSConfig(cfg.Cert, cfg.Key, cfg.CACert)
		if err != nil {
			return nil, err
		}

		return NewTLSConnection(ctx, cfg.URI, tlsConfig, cfg.ID)
	}

	return NewConnection(ctx, cfg.URI, cfg.Device, cfg.Comset, cfg.Baudrate, cfg.Protocol(), cfg.ID)
}

// NewTLSConnection creates a Modbus TCP connection secured with TLS. It
// mirrors NewConnection but always uses the TCP transport wrapped in TLS
// using the supplied tls.Config. The default port is 802 (Modbus/TCP Security).
func NewTLSConnection(ctx context.Context, uri string, tlsConfig *tls.Config, slaveID uint8) (*Connection, error) {
	conn, err := physicalTLSConnection(ctx, uri, tlsConfig)
	if err != nil {
		return nil, err
	}

	return &Connection{
		slaveID:    slaveID,
		Connection: conn.Clone(slaveID),
		logger:     conn.logger,
	}, nil
}

func physicalTLSConnection(ctx context.Context, uri string, tlsConfig *tls.Config) (*meterConnection, error) {
	uri = util.DefaultPort(uri, 802)

	handler := gridx.NewTCPClientHandler(uri, gridx.WithTLSConfig(tlsConfig))
	handler.LinkRecoveryTimeout = 0
	handler.ProtocolRecoveryTimeout = 0

	conn := &meters.TCP{
		Client:  gridx.NewClient(handler),
		Handler: handler,
	}

	return registeredConnection(ctx, uri, Tls, conn)
}
