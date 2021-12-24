package porsche

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	cv "github.com/nirasan/go-oauth-pkce-code-verifier"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"
)

const (
	OAuthURI = "https://login.porsche.com"
)

// https://login.porsche.com/.well-known/openid-configuration
var (
	OAuth2Config = &oauth2.Config{
		ClientID:    "4mPO3OE5Srjb1iaUGWsbqKBvvesya8oA",
		RedirectURL: "https://my.porsche.com/core/de/de_DE/",
		Endpoint: oauth2.Endpoint{
			AuthURL:  OAuthURI + "/as/authorization.oauth2",
			TokenURL: OAuthURI + "/as/token.oauth2",
		},
		Scopes: []string{"openid"},
	}

	EmobilityOAuth2Config = &oauth2.Config{
		ClientID:    "NJOxLv4QQNrpZnYQbb7mCvdiMxQWkHDq",
		RedirectURL: "https://my.porsche.com/myservices/auth/auth.html",
		Endpoint:    OAuth2Config.Endpoint,
		Scopes:      OAuth2Config.Scopes,
	}
)

type AccessTokens struct {
	Token, EmobilityToken *oauth2.Token
}

// Identity is the Porsche Identity client
type Identity struct {
	log *util.Logger
	*request.Helper
	user, password string
}

// NewIdentity creates Porsche identity
func NewIdentity(log *util.Logger, user, password string) *Identity {
	v := &Identity{
		log:      log,
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
	}

	jar, _ := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	// track cookies and follow all (>10) redirects
	v.Client.Jar = jar
	v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return nil
	}

	return v
}

func (v *Identity) Login() (AccessTokens, error) {
	var accessTokens AccessTokens

	// get the login page to get the cookies for the subsequent requests
	resp, err := v.Client.Get("https://login.porsche.com/auth/de/de_DE")
	if err != nil {
		return accessTokens, err
	}
	resp.Body.Close()

	query, err := url.ParseQuery(resp.Request.URL.RawQuery)
	if err != nil {
		return accessTokens, err
	}

	sec := query.Get("sec")
	resume := query.Get("resume")
	state := query.Get("state")
	thirdPartyID := query.Get("thirdPartyId")

	dataLoginAuth := url.Values{
		"sec":          []string{sec},
		"resume":       []string{resume},
		"thirdPartyId": []string{thirdPartyID},
		"state":        []string{state},
		"username":     []string{v.user},
		"password":     []string{v.password},
		"keeploggedin": []string{"false"},
	}

	req, err := request.New(http.MethodPost, "https://login.porsche.com/auth/api/v1/de/de_DE/public/login", strings.NewReader(dataLoginAuth.Encode()), request.URLEncoding)
	if err != nil {
		return accessTokens, err
	}

	// process the auth so the session is authenticated
	if resp, err = v.Client.Do(req); err != nil {
		return accessTokens, err
	}
	resp.Body.Close()

	// get the token for the generic API
	token, err := v.fetchToken(OAuth2Config)
	if err != nil {
		return accessTokens, err
	}
	accessTokens.Token = token

	token, err = v.fetchToken(EmobilityOAuth2Config)
	if err != nil {
		return accessTokens, err
	}
	accessTokens.EmobilityToken = token

	return accessTokens, err
}

func (v *Identity) fetchToken(oc *oauth2.Config) (*oauth2.Token, error) {
	cv, err := cv.CreateCodeVerifier()
	if err != nil {
		return nil, err
	}

	uri := oc.AuthCodeURL("uvobn7XJs1", oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", cv.CodeChallengeS256()),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("country", "de"),
		oauth2.SetAuthURLParam("locale", "de_DE"),
	)

	resp, err := v.Client.Get(uri)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	query, err := url.ParseQuery(resp.Request.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	code := query.Get("code")
	if code == "" {
		return nil, errors.New("no auth code")
	}

	ctx, cancel := context.WithTimeout(
		context.WithValue(context.Background(), oauth2.HTTPClient, v.Client),
		request.Timeout,
	)
	defer cancel()

	token, err := oc.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", cv.CodeChallengePlain()),
	)

	os.Exit(1)

	return token, err
}

func (v *Identity) FindVehicle(accessTokens AccessTokens, vin string) (string, error) {
	vehiclesURL := "https://api.porsche.com/core/api/v3/de/de_DE/vehicles"
	req, err := request.New(http.MethodGet, vehiclesURL, nil, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", accessTokens.Token.AccessToken),
		"apikey":        OAuth2Config.ClientID,
	})

	if err != nil {
		return "", err
	}

	var vehicles []VehicleResponse
	if err = v.DoJSON(req, &vehicles); err != nil {
		return "", err
	}

	var foundVehicle VehicleResponse

	if vin == "" && len(vehicles) == 1 {
		foundVehicle = vehicles[0]
	} else {
		for _, vehicleItem := range vehicles {
			if vehicleItem.VIN == strings.ToUpper(vin) {
				foundVehicle = vehicleItem
			}
		}
	}
	if foundVehicle.VIN == "" {
		return "", errors.New("vin not found")
	}

	v.log.DEBUG.Printf("found vehicle: %v", foundVehicle.VIN)

	// check if vehicle is paired
	uri := fmt.Sprintf("%s/%s/pairing", vehiclesURL, foundVehicle.VIN)
	req, err = request.New(http.MethodGet, uri, nil, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", accessTokens.Token.AccessToken),
		"apikey":        OAuth2Config.ClientID,
	})

	if err != nil {
		return "", err
	}

	var pairing VehiclePairingResponse
	if err = v.DoJSON(req, &pairing); err != nil {
		return "", err
	}

	if pairing.Status != "PAIRINGCOMPLETE" {
		return "", errors.New("vehicle is not paired with the My Porsche account")
	}

	// now check if we get any response at all for a status request
	// there are PHEV which do not provide any data, even thought they are PHEV
	uri = fmt.Sprintf("https://api.porsche.com/vehicle-data/de/de_DE/status/%s", foundVehicle.VIN)
	req, err = request.New(http.MethodGet, uri, nil, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", accessTokens.Token.AccessToken),
		"apikey":        OAuth2Config.ClientID,
	})

	if err != nil {
		return "", err
	}

	if _, err = v.DoBody(req); err != nil {
		return "", errors.New("vehicle is not capable of providing data")
	}

	return foundVehicle.VIN, err
}
