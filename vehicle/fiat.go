package vehicle

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/andig/evcc/vehicle/fiat"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/aws/aws-sdk-go/service/cognitoidentity"
	"github.com/thoas/go-funk"
)

// https://github.com/TA2k/ioBroker.fiat

// Fiat is an api.Vehicle implementation for Fiat cars
type Fiat struct {
	*embed
	*request.Helper
	log                 *util.Logger
	user, password, vin string
	uid                 string
	// creds               *credentials.Credentials
	creds   *cognitoidentity.Credentials
	statusG func() (interface{}, error)
}

func init() {
	registry.Add("fiat", NewFiatFromConfig)
}

// NewFiatFromConfig creates a new vehicle
func NewFiatFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		Cache               time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, errors.New("missing credentials")
	}

	log := util.NewLogger("fiat")

	v := &Fiat{
		embed:    &cc.embed,
		Helper:   request.NewHelper(log),
		log:      log,
		user:     cc.User,
		password: cc.Password,
		vin:      strings.ToUpper(cc.VIN),
	}

	err := v.login()
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	if cc.VIN == "" {
		v.vin, err = findVehicle(v.vehicles())
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", v.vin)
		}
	}

	v.statusG = provider.NewCached(func() (interface{}, error) {
		return v.status()
	}, cc.Cache).InterfaceGetter()

	return v, err
}

// login authenticates with username/password to get new aws credentials
func (v *Fiat) login() error {
	v.Client.Jar, _ = cookiejar.New(nil)

	uri := fmt.Sprintf("%s/accounts.webSdkBootstrap", fiat.LoginURI)

	data := url.Values(map[string][]string{
		"APIKey":   {fiat.ApiKey},
		"pageURL":  {"https://myuconnect.fiat.com/de/de/vehicle-services"},
		"sdk":      {"js_latest"},
		"sdkBuild": {"12234"},
		"format":   {"json"},
	})

	headers := map[string]string{
		"Accept": "*/*",
	}

	req, err := request.New(http.MethodGet, uri, nil, headers)
	if err == nil {
		req.URL.RawQuery = data.Encode()
		_, err = v.Do(req)
	}

	var res struct {
		ErrorCode    int
		UID          string
		StatusReason string
		SessionInfo  struct {
			LoginToken string `json:"login_token"`
			ExpiresIn  string `json:"expires_in"`
		}
	}

	if err == nil {
		uri = fmt.Sprintf("%s/accounts.login", fiat.LoginURI)

		data := url.Values(map[string][]string{
			"loginID":           {v.user},
			"password":          {v.password},
			"sessionExpiration": {"7776000"},
			"APIKey":            {fiat.ApiKey},
			"pageURL":           {"https://myuconnect.fiat.com/de/de/login"},
			"sdk":               {"js_latest"},
			"sdkBuild":          {"12234"},
			"format":            {"json"},
			"targetEnv":         {"jssdk"},
			"include":           {"profile,data,emails"}, // subscriptions,preferences
			"includeUserInfo":   {"true"},
			"loginMode":         {"standard"},
			"lang":              {"de0de"},
			"source":            {"showScreenSet"},
			"authMode":          {"cookie"},
		})

		headers := map[string]string{
			"Accept":       "*/*",
			"Content-type": "application/x-www-form-urlencoded",
		}

		if req, err = request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), headers); err == nil {
			if err = v.DoJSON(req, &res); err == nil {
				v.uid = res.UID
			}
		}
	}

	var token struct {
		ErrorCode    int `json:"errorCode"`
		StatusReason string
		IDToken      string `json:"id_token"`
	}

	if err == nil {
		uri = fmt.Sprintf("%s/accounts.getJWT", fiat.LoginURI)

		data := url.Values(map[string][]string{
			"fields":      {"profile.firstName,profile.lastName,profile.email,country,locale,data.disclaimerCodeGSDP"}, // data.GSDPisVerified
			"APIKey":      {fiat.ApiKey},
			"pageURL":     {"https://myuconnect.fiat.com/de/de/dashboard"},
			"sdk":         {"js_latest"},
			"sdkBuild":    {"12234"},
			"format":      {"json"},
			"login_token": {res.SessionInfo.LoginToken},
			"authMode":    {"cookie"},
		})

		headers := map[string]string{
			"Accept": "*/*",
		}

		if req, err = request.New(http.MethodGet, uri, nil, headers); err == nil {
			req.URL.RawQuery = data.Encode()
			err = v.DoJSON(req, &token)
		}
	}

	var identity struct {
		Token, IdentityID string
	}

	if err == nil {
		uri = "https://authz.sdpr-01.fcagcv.com/v2/cognito/identity/token"

		data := struct {
			GigyaToken string `json:"gigya_token"`
		}{
			GigyaToken: token.IDToken,
		}

		headers := map[string]string{
			"Content-type":        "application/json",
			"X-Clientapp-Version": "1.0",
			"ClientRequestId":     util.RandomString(16),
			"X-api-key":           "qLYupk65UU1tw2Ih1cJhs4izijgRDbir2UFHA3Je",
			"X-originator-type":   "web",
		}

		if req, err = request.New(http.MethodPost, uri, request.MarshalJSON(data), headers); err == nil {
			err = v.DoJSON(req, &identity)
		}
	}

	var credRes *cognitoidentity.GetCredentialsForIdentityOutput

	if err == nil {
		session := session.Must(session.NewSession(&aws.Config{Region: aws.String("eu-west-1")}))
		svc := cognitoidentity.New(session)

		credRes, err = svc.GetCredentialsForIdentity(&cognitoidentity.GetCredentialsForIdentityInput{
			IdentityId: &identity.IdentityID,
			Logins: map[string]*string{
				"cognito-identity.amazonaws.com": &identity.Token,
			},
		})
	}

	if err == nil {
		v.creds = credRes.Credentials
	}

	return err
}

func (v *Fiat) request(method, uri string, body io.Reader) (*http.Request, error) {
	// refresh credentials
	if v.creds.Expiration.After(time.Now().Add(-time.Minute)) {
		if err := v.login(); err != nil {
			return nil, err
		}
	}

	headers := map[string]string{
		"Content-Type":        "application/json",
		"x-clientapp-version": "1.0",
		"clientrequestid":     util.RandomString(16),
		"X-Api-Key":           "qLYupk65UU1tw2Ih1cJhs4izijgRDbir2UFHA3Je",
		"x-originator-type":   "web",
	}

	req, err := request.New(method, uri, body, headers)
	if err == nil {
		signer := v4.NewSigner(credentials.NewStaticCredentials(
			*v.creds.AccessKeyId, *v.creds.SecretKey, *v.creds.SessionToken,
		))
		_, err = signer.Sign(req, nil, "execute-api", "eu-west-1", time.Now())
	}

	return req, err
}

func (v *Fiat) vehicles() ([]string, error) {
	var res struct {
		Vehicles []fiat.Vehicle
	}

	uri := fmt.Sprintf("%s/v4/accounts/%s/vehicles?stage=ALL", fiat.ApiURI, v.uid)

	req, err := v.request(http.MethodGet, uri, nil)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	vehicles := funk.Map(res.Vehicles, func(v fiat.Vehicle) string {
		return v.VIN
	}).([]string)

	return vehicles, err
}

func (v *Fiat) status() (interface{}, error) {
	var res fiat.Status

	uri := fmt.Sprintf("%s/v2/accounts/%s/vehicles/%s/status", fiat.ApiURI, v.uid, v.vin)

	req, err := v.request(http.MethodGet, uri, nil)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}

// SoC implements the api.Vehicle interface
func (v *Fiat) SoC() (float64, error) {
	res, err := v.statusG()
	if res, ok := res.(fiat.Status); err == nil && ok {
		return float64(res.EvInfo.Battery.StateOfCharge), nil
	}

	return 0, err
}

var _ api.VehicleRange = (*Fiat)(nil)

// Range implements the api.VehicleRange interface
func (v *Fiat) Range() (int64, error) {
	res, err := v.statusG()
	if res, ok := res.(fiat.Status); err == nil && ok {
		return int64(res.EvInfo.Battery.DistanceToEmpty.Value), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Fiat)(nil)

// Status implements the api.ChargeState interface
func (v *Fiat) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if res, ok := res.(fiat.Status); err == nil && ok {
		if res.EvInfo.Battery.PlugInStatus {
			status = api.StatusB // connected, not charging
		}
		if res.EvInfo.Battery.ChargingStatus == "CHARGING" {
			status = api.StatusC // charging
		}
	}

	return status, err
}
