package cloud

import (
	_ "embed"

	"google.golang.org/grpc"
)

var Host = "sponsor.evcc.io:8080"

var conn *grpc.ClientConn

func Connection(hostPort string) (*grpc.ClientConn, error) {
	if conn != nil {
		return conn, nil
	}

	var err error
	conn, err = grpc.Dial(hostPort)

	return conn, err
}
