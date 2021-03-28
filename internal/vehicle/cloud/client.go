package cloud

import (
	"os"

	"github.com/andig/evcc/proto/pb"
	"github.com/andig/evcc/util"
	"google.golang.org/grpc"
)

var Host = os.Getenv("GRPC_URI")

var (
	conn   *grpc.ClientConn
	client pb.VehicleClient
)

func Client(log *util.Logger, uri string) (pb.VehicleClient, error) {
	var err error
	if conn == nil {
		conn, err = grpc.Dial(uri, grpc.WithInsecure())
	}

	if client == nil && err == nil {
		client = pb.NewVehicleClient(conn)
	}

	return client, err
}
