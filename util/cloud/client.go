package cloud

import (
	"crypto/tls"
	_ "embed"

	"github.com/evcc-io/evcc/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	host = util.Getenv("GRPC_URI", "sponsor.evcc.io:8080")

	conn *grpc.ClientConn
)

func Connection() (*grpc.ClientConn, error) {
	if conn != nil {
		return conn, nil
	}

	var err error
	creds := credentials.NewTLS(new(tls.Config))
	conn, err = grpc.Dial(host, grpc.WithTransportCredentials(creds))

	return conn, err
}
