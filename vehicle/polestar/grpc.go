package polestar

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/polestar/pb"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Polestar gRPC battery API.
//
// Polestar removed the chargingStatus field from the GraphQL API. The gRPC
// battery service still exposes the charger connection and charging status.
// Protocol reconstructed by the pypolestar project, see
// https://github.com/pypolestar/pypolestar.
const (
	// c3DiscoveryURI resolves the dynamic gRPC host of the battery service
	c3DiscoveryURI    = "https://cnepmob.volvocars.com"
	c3DiscoveryAccept = "application/volvo.cloud.cnepmob.v1+json"

	// defaultGrpcPort is used when discovery does not return a port
	defaultGrpcPort = 443
)

// GrpcAPI is the Polestar gRPC battery API client
type GrpcAPI struct {
	client pb.BatteryServiceClient
}

// NewGrpcAPI creates a Polestar gRPC battery API client. The token source is
// shared with the GraphQL API so both reuse the same OAuth2 session.
func NewGrpcAPI(log *util.Logger, ts oauth2.TokenSource) (*GrpcAPI, error) {
	host, err := discoverC3Host(log)
	if err != nil {
		return nil, fmt.Errorf("c3 discovery: %w", err)
	}

	conn, err := grpc.NewClient(host,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
		grpc.WithPerRPCCredentials(tokenCredentials{ts}),
	)
	if err != nil {
		return nil, err
	}

	return &GrpcAPI{
		client: pb.NewBatteryServiceClient(conn),
	}, nil
}

// tokenCredentials adapts an oauth2.TokenSource to grpc.PerRPCCredentials,
// adding the bearer token as gRPC authorization metadata on every call.
type tokenCredentials struct {
	oauth2.TokenSource
}

func (c tokenCredentials) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	token, err := c.Token()
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"authorization": token.Type() + " " + token.AccessToken,
	}, nil
}

func (tokenCredentials) RequireTransportSecurity() bool {
	return true
}

// discoverC3Host resolves the C3 gRPC host:port via the cnepmob discovery endpoint
func discoverC3Host(log *util.Logger) (string, error) {
	req, err := request.New(http.MethodGet, c3DiscoveryURI, nil, map[string]string{
		"Accept": c3DiscoveryAccept,
	})
	if err != nil {
		return "", err
	}

	var res struct {
		C3 struct {
			GrpcHost string `json:"grpcHost"`
			GrpcPort int    `json:"grpcPort"`
		} `json:"c3"`
	}
	if err := request.NewHelper(log).DoJSON(req, &res); err != nil {
		return "", err
	}

	if res.C3.GrpcHost == "" {
		return "", errors.New("missing grpc host")
	}

	port := res.C3.GrpcPort
	if port == 0 {
		port = defaultGrpcPort
	}

	return net.JoinHostPort(res.C3.GrpcHost, strconv.Itoa(port)), nil
}

// Battery returns the latest battery state for the given VIN
func (v *GrpcAPI) Battery(ctx context.Context, vin string) (*pb.Battery, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, "vin", vin)

	res, err := v.client.GetLatestBattery(ctx, &pb.GetBatteryRequest{
		Id:  uuid.NewString(),
		Vin: vin,
	})
	if err != nil {
		return nil, retryable(err)
	}

	battery := res.GetBattery()
	if battery == nil {
		return nil, api.ErrNotAvailable
	}

	return battery, nil
}

// retryable marks transient gRPC errors as api.ErrMustRetry so util.Cached
// retries them on the next cycle instead of caching the failure and delaying
// recovery via exponential back-off.
func retryable(err error) error {
	switch status.Code(err) {
	case codes.Unavailable, codes.DeadlineExceeded:
		return fmt.Errorf("%w: %w", api.ErrMustRetry, err)
	default:
		return err
	}
}
