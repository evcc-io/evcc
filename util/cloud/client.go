package cloud

import (
	"crypto/tls"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var Host = "cloud.evcc.io:8080"

var (
	conn *grpc.ClientConn
)

func Connection(uri string) (*grpc.ClientConn, error) {
	var err error
	if conn == nil {
		transportOption := grpc.WithInsecure()
		if !strings.HasPrefix(uri, "localhost") {
			transportOption = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
				InsecureSkipVerify: true,
			}))
		}
		conn, err = grpc.Dial(uri, transportOption)
	}

	return conn, err
}
