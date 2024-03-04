package cloud

import (
	_ "embed"

	"github.com/evcc-io/evcc/util"
	"google.golang.org/grpc"
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
	conn, err = grpc.Dial(host)

	return conn, err
}
