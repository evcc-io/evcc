package cloud

import (
	"crypto/tls"
	_ "embed"
	"net"
	"time"

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
	// quiesce idle connections instead of churning against the server's idle close
	conn, err = grpc.NewClient(hostport,
		grpc.WithTransportCredentials(creds),
		grpc.WithIdleTimeout(60*time.Second),
	)

	return conn, err
}
