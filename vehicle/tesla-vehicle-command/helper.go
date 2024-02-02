package vc

import (
	"errors"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/golang-jwt/jwt/v5"
	"github.com/teslamotors/vehicle-command/pkg/connector/inet"
	"golang.org/x/oauth2"
)

// apiError converts HTTP 408 error to ErrTimeout
func apiError(err error) error {
	if err != nil && (errors.Is(err, inet.ErrVehicleNotAwake) ||
		strings.HasSuffix(err.Error(), "408 Request Timeout") || strings.HasSuffix(err.Error(), "408 (Request Timeout)")) {
		err = api.ErrAsleep
	}
	return err
}

func SettingsKey(subject string) string {
	return fmt.Sprintf("tesla-command.%s", subject)
}

func TokenClaims(token *oauth2.Token) (*jwt.RegisteredClaims, error) {
	var claims jwt.RegisteredClaims
	if _, _, err := jwt.NewParser().ParseUnverified(token.AccessToken, &claims); err != nil {
		return nil, err
	}
	return &claims, nil
}
