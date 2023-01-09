package cloud

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"errors"
	"fmt"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var Host = "sponsor.evcc.io:8080"

var conn *grpc.ClientConn

//go:embed ca-cert.pem
var caCert []byte

func caPEM() []byte {
	copy := bytes.NewBuffer(caCert)
	return copy.Bytes()
}

func loadTLSCredentials() (*tls.Config, error) {
	certPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	if !certPool.AppendCertsFromPEM(caPEM()) {
		return nil, fmt.Errorf("failed to add CA certificate")
	}

	// create the credentials and return it
	config := &tls.Config{
		RootCAs: certPool,
	}

	return config, nil
}

func verifyConnection(host string) func(conn tls.ConnectionState) error {
	return func(conn tls.ConnectionState) error {
		if len(conn.PeerCertificates) > 0 {
			peer := conn.PeerCertificates[0]
			return peer.VerifyHostname(host)
		}

		return errors.New("missing host certificate")
	}
}

func Connection(hostPort string) (*grpc.ClientConn, error) {
	var err error
	if conn == nil {
		creds := insecure.NewCredentials()

		if !strings.HasPrefix(hostPort, "localhost") {
			host, _, err := net.SplitHostPort(hostPort)
			if err != nil {
				return nil, err
			}

			var tlsConfig *tls.Config
			if tlsConfig, err = loadTLSCredentials(); err != nil {
				return nil, err
			}

			// make sure it matches the hostname
			tlsConfig.VerifyConnection = verifyConnection(host)

			creds = credentials.NewTLS(tlsConfig)
		}
		conn, err = grpc.Dial(hostPort, grpc.WithTransportCredentials(creds))
	}

	return conn, err
}
