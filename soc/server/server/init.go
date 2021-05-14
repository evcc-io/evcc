package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/soc/cert/ca"
	"github.com/andig/evcc/soc/cert/server"
	"github.com/andig/evcc/soc/proto/pb"
	"github.com/andig/evcc/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	port      = util.Getenv("GRPC_PORT", "8080")
	tlsConfig *tls.Config
)

func init() {
	var err error
	if tlsConfig, err = loadTLSCredentials(); err != nil {
		log.Fatalf("cannot load TLS credentials: %v", err)
	}

	registerMetrics()
}

func loadTLSCredentials() (*tls.Config, error) {
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(ca.PEM()) {
		return nil, fmt.Errorf("failed to add client CA's certificate")
	}

	// Load server's certificate and private key
	serverCert, err := server.Certificate()
	if err != nil {
		return nil, err
	}

	// Create the credentials and return it
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.VerifyClientCertIfGiven,
		ClientCAs:    certPool,
	}

	return config, nil
}

func Run() {
	log.Println("grpc:", ":"+port)

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	serverOptions := []grpc.ServerOption{}
	if tlsConfig != nil {
		serverOptions = append(serverOptions, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}
	grpcServer := grpc.NewServer(serverOptions...)

	pb.RegisterVehicleServer(grpcServer, &VehicleServer{
		vehicles: make(map[string]map[int64]api.Vehicle),
	})
	pb.RegisterAuthServer(grpcServer, &AuthServer{})

	log.Fatal(grpcServer.Serve(listener))
}
