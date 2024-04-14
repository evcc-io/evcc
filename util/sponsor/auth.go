package sponsor

import (
	"context"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api/proto/pb"
	"github.com/evcc-io/evcc/util/cloud"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	Subject, Token string
	ExpiresAt      time.Time
)

const unavailable = "sponsorship unavailable"

func IsAuthorized() bool {
	return len(Subject) > 0
}

func IsAuthorizedForApi() bool {
	return IsAuthorized() && Subject != unavailable
}

// check and set sponsorship token
func ConfigureSponsorship(token string) error {
	if token == "" {
		var err error
		if token, err = readSerial(); token == "" || err != nil {
			return err
		}
	}

	conn, err := cloud.Connection()
	if err != nil {
		return err
	}

	client := pb.NewAuthClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := client.IsAuthorized(ctx, &pb.AuthRequest{Token: token})
	if err == nil && res.Authorized {
		Subject = res.Subject
		ExpiresAt = res.ExpiresAt.AsTime()
		Token = token
	}

	if err != nil {
		if s, ok := status.FromError(err); ok && s.Code() != codes.Unknown {
			Subject = unavailable
			err = nil
		} else {
			err = fmt.Errorf("sponsortoken: %w", err)
		}
	}

	return err
}

type StatusStruct struct {
	IsAuthorized       bool      `json:"isAuthorized"`
	IsAuthorizedForApi bool      `json:"isAuthorizedForApi"`
	Subject            string    `json:"subject"`
	ExpiresAt          time.Time `json:"expires"`
	ExpiresIn          int64     `json:"expiresIn,omitempty"`
}

// Status returns the sponsorship status
func Status() StatusStruct {
	var expiresIn int64
	if IsAuthorizedForApi() {
		expiresIn = int64(time.Until(ExpiresAt).Seconds())
	}

	return StatusStruct{
		IsAuthorized:       IsAuthorized(),
		IsAuthorizedForApi: IsAuthorizedForApi(),
		Subject:            Subject,
		ExpiresAt:          ExpiresAt,
		ExpiresIn:          expiresIn,
	}
}
