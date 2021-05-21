package cloud

import (
	"google.golang.org/grpc"
)

var Host = "cloud.evcc.io:8080"

var (
	conn *grpc.ClientConn
)

func Connection(uri string) (*grpc.ClientConn, error) {
	var err error
	if conn == nil {
		transportOption := grpc.WithInsecure()
		conn, err = grpc.Dial(uri, transportOption)
	}

	return conn, err
}
