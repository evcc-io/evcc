package fiat

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/aws/aws-sdk-go/service/cognitoidentity"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/samber/lo"
)

const (
	LoginURI = "https://loginmyuconnect.fiat.com"
	TokenURI = "https://authz.sdpr-01.fcagcv.com/v2/cognito/identity/token"

	Region = "eu-west-1"
)

type Identity struct {
	*request.Helper
	user, password string
	uid            string
	creds          *cognitoidentity.Credentials
}

// NewIdentity creates Fiat identity
func NewIdentity(log *util.Logger, user, password string) *Identity {
	return &Identity{
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
	}
}

// Login authenticates with username/password to get new aws credentials
func (v *Identity) Login() error {
	v.Client.Jar, _ = cookiejar.New(nil)

	uri := fmt.Sprintf("%s/accounts.webSdkBootstrap", LoginURI)

	data := url.Values{
		"APIKey":   {ApiKey},
		"pageURL":  {"https://myuconnect.fiat.com/de/de/vehicle-services"},
		"sdk":      {"js_latest"},
		"sdkBuild": {"12234"},
		"format":   {"json"},
	}

	headers := map[string]string{
		"Accept": "*/*",
	}

	req, err := request.New(http.MethodGet, uri, nil, headers)
	if err == nil {
		req.URL.RawQuery = data.Encode()

		var resp *http.Response
		if resp, err = v.Do(req); err == nil {
			resp.Body.Close()
		}
	}

	var res struct {
		ErrorInfo
		UID          string
		StatusReason string
		SessionInfo  struct {
			LoginToken string `json:"login_token"`
			ExpiresIn  string `json:"expires_in"`
		}
	}

	if err == nil {
		uri = fmt.Sprintf("%s/accounts.login", LoginURI)

		data := url.Values{
			"loginID":           {v.user},
			"password":          {v.password},
			"sessionExpiration": {"7776000"},
			"APIKey":            {ApiKey},
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
		}

		headers := map[string]string{
			"Accept":       "*/*",
			"Content-Type": "application/x-www-form-urlencoded",
		}

		if req, err = request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), headers); err == nil {
			if err = v.DoJSON(req, &res); err == nil {
				err = res.ErrorInfo.Error()
				v.uid = res.UID
			}
		}
	}

	var token struct {
		ErrorInfo
		StatusReason string
		IDToken      string `json:"id_token"`
	}

	if err == nil {
		uri = fmt.Sprintf("%s/accounts.getJWT", LoginURI)

		data := url.Values{
			"fields":      {"profile.firstName,profile.lastName,profile.email,country,locale,data.disclaimerCodeGSDP"}, // data.GSDPisVerified
			"APIKey":      {ApiKey},
			"pageURL":     {"https://myuconnect.fiat.com/de/de/dashboard"},
			"sdk":         {"js_latest"},
			"sdkBuild":    {"12234"},
			"format":      {"json"},
			"login_token": {res.SessionInfo.LoginToken},
			"authMode":    {"cookie"},
		}

		headers := map[string]string{
			"Accept": "*/*",
		}

		if req, err = request.New(http.MethodGet, uri, nil, headers); err == nil {
			req.URL.RawQuery = data.Encode()
			if err = v.DoJSON(req, &token); err == nil {
				err = token.ErrorInfo.Error()
			}
		}
	}

	var identity struct {
		Token, IdentityID string
	}

	if err == nil {
		data := struct {
			GigyaToken string `json:"gigya_token"`
		}{
			GigyaToken: token.IDToken,
		}

		headers := map[string]string{
			"Content-Type":        "application/json",
			"X-Clientapp-Version": "1.0",
			"ClientRequestId":     lo.RandomString(16, lo.LettersCharset),
			"X-Api-Key":           XApiKey,
			"X-Originator-Type":   "web",
		}

		if req, err = request.New(http.MethodPost, TokenURI, request.MarshalJSON(data), headers); err == nil {
			err = v.DoJSON(req, &identity)
		}
	}

	var credRes *cognitoidentity.GetCredentialsForIdentityOutput

	if err == nil {
		session := session.Must(session.NewSession(&aws.Config{Region: aws.String(Region)}))
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

// UID returns the logged in users uid
func (v *Identity) UID() string {
	return v.uid
}

// Sign signs an AWS request using identity's credentials
func (v *Identity) Sign(req *http.Request, body io.ReadSeeker) error {
	// refresh credentials
	if v.creds.Expiration.Before(time.Now().Add(-time.Minute)) {
		if err := v.Login(); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
	}

	// sign request
	signer := v4.NewSigner(credentials.NewStaticCredentials(
		*v.creds.AccessKeyId, *v.creds.SecretKey, *v.creds.SessionToken,
	))
	_, err := signer.Sign(req, body, "execute-api", Region, time.Now())

	return err
}
