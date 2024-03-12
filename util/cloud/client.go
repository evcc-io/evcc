package cloud

import (
	"crypto/tls"
	_ "embed"
	"net"

	"github.com/evcc-io/evcc/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	hostport = util.Getenv("GRPC_URI", "sponsor.evcc.io:8080")

	conn *grpc.ClientConn
)

func Connection() (*grpc.ClientConn, error) {
	if conn != nil {
		return conn, nil
	}

	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return nil, err
	}

	creds := credentials.NewTLS(&tls.Config{
		ServerName: host,
	})
	conn, err = grpc.Dial(hostport, grpc.WithTransportCredentials(creds))

	return conn, err
}
