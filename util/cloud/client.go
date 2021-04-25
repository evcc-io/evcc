package cloud

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var Host = "cloud.evcc.io:8080"

var (
	conn *grpc.ClientConn
)

func loadTLSCredentials() (*tls.Config, error) {
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caPEM()) {
		return nil, fmt.Errorf("failed to add CA certificate")
	}

	// create the credentials and return it
	config := &tls.Config{
		RootCAs: certPool,
	}

	return config, nil
}

func Connection(uri string) (*grpc.ClientConn, error) {
	var err error
	if conn == nil {
		var tlsConfig *tls.Config
		if tlsConfig, err = loadTLSCredentials(); err != nil {
			return nil, fmt.Errorf("cannot load TLS credentials: %w", err)
		}

		transportOption := grpc.WithInsecure()
		if tlsConfig != nil {
			transportOption = grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
		}

		conn, err = grpc.Dial(uri, transportOption)
	}

	return conn, err
}
