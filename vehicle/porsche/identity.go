package porsche

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	cv "github.com/nirasan/go-oauth-pkce-code-verifier"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"
)

const (
	ClientID          = "4mPO3OE5Srjb1iaUGWsbqKBvvesya8oA"
	EmobilityClientID = "NJOxLv4QQNrpZnYQbb7mCvdiMxQWkHDq"
)

type AccessTokens struct {
	Token, EmobilityToken oauth2.Token
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
	var token oauth.Token
	token, err = v.fetchToken(false)
	if err != nil {
		return accessTokens, err
	}
	accessTokens.Token = (oauth2.Token)(token)

	token, err = v.fetchToken(true)
	if err != nil {
		return accessTokens, err
	}
	accessTokens.EmobilityToken = (oauth2.Token)(token)

	return accessTokens, err
}

func (v *Identity) fetchToken(emobility bool) (oauth.Token, error) {
	var pr oauth.Token

	actualClientID := ClientID
	redirectURI := "https://my.porsche.com/core/de/de_DE/"

	if emobility {
		actualClientID = EmobilityClientID
		redirectURI = "https://my.porsche.com/myservices/auth/auth.html"
	}

	var CodeVerifier, _ = cv.CreateCodeVerifier()
	codeChallenge := CodeVerifier.CodeChallengeS256()

	dataTokenAuth := url.Values{
		"redirect_uri":          []string{redirectURI},
		"client_id":             []string{actualClientID},
		"response_type":         []string{"code"},
		"state":                 []string{"uvobn7XJs1"},
		"scope":                 []string{"openid"},
		"access_type":           []string{"offline"},
		"country":               []string{"de"},
		"locale":                []string{"de_DE"},
		"code_challenge":        []string{codeChallenge},
		"code_challenge_method": []string{"S256"},
	}

	req, err := http.NewRequest(http.MethodGet, "https://login.porsche.com/as/authorization.oauth2", nil)
	if err != nil {
		return pr, err
	}

	req.URL.RawQuery = dataTokenAuth.Encode()

	resp, err := v.Client.Do(req)
	if err != nil {
		return pr, err
	}
	resp.Body.Close()

	query, err := url.ParseQuery(resp.Request.URL.RawQuery)
	if err != nil {
		return pr, err
	}

	authCode := query.Get("code")
	if authCode == "" {
		return pr, errors.New("no auth code")
	}

	codeVerifier := CodeVerifier.CodeChallengePlain()

	dataAPIToken := url.Values{
		"grant_type":    []string{"authorization_code"},
		"client_id":     []string{actualClientID},
		"redirect_uri":  []string{redirectURI},
		"code":          []string{authCode},
		"code_verifier": []string{codeVerifier},
	}

	req, err = request.New(http.MethodPost, "https://login.porsche.com/as/token.oauth2", strings.NewReader(dataAPIToken.Encode()), request.URLEncoding)
	if err == nil {
		err = v.DoJSON(req, &pr)
	}

	if pr.AccessToken == "" {
		return pr, errors.New("could not obtain token")
	}

	return pr, err
}

func (v *Identity) FindVehicle(accessTokens AccessTokens, vin string) (string, error) {
	vehiclesURL := "https://api.porsche.com/core/api/v3/de/de_DE/vehicles"
	req, err := request.New(http.MethodGet, vehiclesURL, nil, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", accessTokens.Token.AccessToken),
		"apikey":        ClientID,
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
		"apikey":        ClientID,
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
		"apikey":        ClientID,
	})

	if err != nil {
		return "", err
	}

	if _, err = v.DoBody(req); err != nil {
		return "", errors.New("vehicle is not capable of providing data")
	}

	return foundVehicle.VIN, err
}
