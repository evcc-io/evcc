package device

import (
	"context"
	"fmt"
	"os"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
)

const (
	deviceToken = "deviceToken"
	githubUser  = "githubUser"
)

var gh = &oauth2.Config{
	// local testing credentials
	ClientID:     "Ov23ctyJ4CnoVVOUpPsM",
	ClientSecret: "1bd1bfbc76c7225a6e26d1c1fae1886ceb29519c",
	Endpoint: oauth2.Endpoint{
		AuthURL:       "https://github.com/login/oauth/authorize",
		TokenURL:      "https://github.com/login/oauth/access_token",
		DeviceAuthURL: "https://github.com/login/device/code",
	},
}

func init() {
	if v := os.Getenv("CLIENT_ID"); v != "" {
		gh.ClientID = v
	}
	if v := os.Getenv("CLIENT_SECRET"); v != "" {
		gh.ClientSecret = v
	}
}

func waitForAuthorization(da *oauth2.DeviceAuthResponse) {
	// "pending"
	token, err := gh.DeviceAccessToken(context.Background(), da)
	if err != nil {
		// "failed"
		fmt.Println("DeviceAccessToken:", err)
		settings.SetString(deviceToken, "")
		return
	}

	// "done"
	settings.SetString(deviceToken, token.AccessToken)

	// TODO logging
	fmt.Println("DeviceAccessToken:", token)

	ghClient := github.NewClient(gh.Client(context.Background(), token))
	user, _, err := ghClient.Users.Get(context.Background(), "")
	if err != nil {
		settings.SetString(githubUser, "")
		fmt.Println("github user:", err)
		return
	}

	settings.SetString(githubUser, *user.Login)

	// TODO state update
}

func Authorize() (*oauth2.DeviceAuthResponse, error) {
	// "pending"
	da, err := gh.DeviceAuth(context.TODO())

	// {
	// 	"expires_in": 898,
	// 	"device_code": "7152a6139620b9918814d3d0c66adaa26e3e38f2",
	// 	"user_code": "3B2E-6216",
	// 	"verification_uri": "https://github.com/login/device",
	// 	"interval": 5
	// }

	if err == nil {
		go waitForAuthorization(da)
	} else {
		// "failed"
		_ = 1
	}

	return da, err
}
