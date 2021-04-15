package cloud

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/andig/evcc/soc/proto/pb"
	"github.com/andig/evcc/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var Host = "cloud.evcc.io:8080"

var (
	conn   *grpc.ClientConn
	client pb.VehicleClient
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

func Client(log *util.Logger, uri string) (pb.VehicleClient, error) {
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

	if client == nil && err == nil {
		client = pb.NewVehicleClient(conn)
	}

	return client, err
}
