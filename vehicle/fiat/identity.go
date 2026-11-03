package fiat

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/samber/lo"
)

const (
	LoginURI   = "https://loginmyuconnect.fiat.com"
	TokenURI   = "https://authz.sdpr-01.fcagcv.com/v2/cognito/identity/token"
	CognitoURI = "https://cognito-identity." + Region + ".amazonaws.com/"

	Region = "eu-west-1"
)

// credentials are the temporary aws credentials issued by cognito
type credentials struct {
	AccessKeyId, SecretKey, SessionToken string
	Expiration                           float64 // unix seconds
}

type Identity struct {
	*request.Helper
	user, password string
	uid            string
	creds          *credentials
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

	data := url.Values{
		"APIKey":   {ApiKey},
		"pageURL":  {"https://myuconnect.fiat.com/de/de/vehicle-services"},
		"sdk":      {"js_latest"},
		"sdkBuild": {"12234"},
		"format":   {"json"},
	}

	uri := fmt.Sprintf("%s/accounts.webSdkBootstrap?%s", LoginURI, data.Encode())
	req, _ := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept": "*/*",
	})
	if _, err := v.Do(req); err != nil {
		return err
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

	uri = fmt.Sprintf("%s/accounts.login", LoginURI)

	data = url.Values{
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

	req, _ = request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), map[string]string{
		"Accept":       "*/*",
		"Content-Type": "application/x-www-form-urlencoded",
	})
	if err := v.DoJSON(req, &res); err != nil {
		return err
	}
	if err := res.ErrorInfo.Error(); err != nil {
		return err
	}

	v.uid = res.UID

	var token struct {
		ErrorInfo
		StatusReason string
		IDToken      string `json:"id_token"`
	}

	data = url.Values{
		"fields":      {"profile.firstName,profile.lastName,profile.email,country,locale,data.disclaimerCodeGSDP"}, // data.GSDPisVerified
		"APIKey":      {ApiKey},
		"pageURL":     {"https://myuconnect.fiat.com/de/de/dashboard"},
		"sdk":         {"js_latest"},
		"sdkBuild":    {"12234"},
		"format":      {"json"},
		"login_token": {res.SessionInfo.LoginToken},
		"authMode":    {"cookie"},
	}

	uri = fmt.Sprintf("%s/accounts.getJWT?%s", LoginURI, data.Encode())

	req, _ = request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept": "*/*",
	})
	if err := v.DoJSON(req, &token); err != nil {
		return err
	}
	if err := token.ErrorInfo.Error(); err != nil {
		return err
	}

	var identity struct {
		Token, IdentityID string
	}

	gigya := struct {
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

	req, _ = request.New(http.MethodPost, TokenURI, request.MarshalJSON(gigya), headers)
	if err := v.DoJSON(req, &identity); err != nil {
		return err
	}

	// GetCredentialsForIdentity is an unsigned json call, no sdk needed
	var creds struct {
		Credentials credentials
	}

	req, _ = request.New(http.MethodPost, CognitoURI, request.MarshalJSON(struct {
		IdentityId string
		Logins     map[string]string
	}{
		IdentityId: identity.IdentityID,
		Logins:     map[string]string{"cognito-identity.amazonaws.com": identity.Token},
	}), map[string]string{
		"Content-Type": "application/x-amz-json-1.1",
		"X-Amz-Target": "AWSCognitoIdentityService.GetCredentialsForIdentity",
	})
	if err := v.DoJSON(req, &creds); err != nil {
		return err
	}

	v.creds = &creds.Credentials
	return nil
}

// UID returns the logged in users uid
func (v *Identity) UID() string {
	return v.uid
}

// Sign signs an AWS request using identity's credentials
func (v *Identity) Sign(req *http.Request, body io.ReadSeeker) error {
	// refresh credentials
	if v.creds == nil || time.Unix(int64(v.creds.Expiration), 0).Before(time.Now().Add(-time.Minute)) {
		if err := v.Login(); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
	}

	payloadHash, err := hashBody(body)
	if err != nil {
		return err
	}

	signV4(req, v.creds.AccessKeyId, v.creds.SecretKey, v.creds.SessionToken, Region, "execute-api", payloadHash, time.Now())
	return nil
}

func hashBody(body io.ReadSeeker) (string, error) {
	if body == nil {
		// For empty payloads, use the SHA-256 hash of empty string
		return "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", nil
	}

	// Read the body content
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}

	// Reset the body seeker to the beginning
	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	// Calculate SHA-256 hash
	hash := sha256.Sum256(bodyBytes)
	return hex.EncodeToString(hash[:]), nil
}
