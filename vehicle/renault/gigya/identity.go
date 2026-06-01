package gigya

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/renault/keys"
)

const (
	errTFAPending           = 403101
	errAccountPendingTFA    = 403102
	providerGigyaEmail      = "gigyaEmail"
	verificationCodeMessage = "Enter the verification code from your Renault email."
)

// Response is a Gigya API response used by Renault login endpoints.
type Response struct {
	ErrorCode         int           `json:"errorCode"`
	ErrorMessage      string        `json:"errorMessage"`
	ErrorDetails      string        `json:"errorDetails"`
	RegToken          string        `json:"regToken"`
	GigyaAssertion    string        `json:"gigyaAssertion"`
	PHVToken          string        `json:"phvToken"`
	ProviderAssertion string        `json:"providerAssertion"`
	SessionInfo       SessionInfo   `json:"sessionInfo"`
	IDToken           string        `json:"id_token"`
	Data              Data          `json:"data"`
	Emails            []EmailRecord `json:"emails"`
}

// SessionInfo contains Gigya session cookie data.
type SessionInfo struct {
	CookieValue string `json:"cookieValue"`
}

// Data contains Renault account metadata stored in Gigya.
type Data struct {
	PersonID string `json:"personId"`
}

// EmailRecord describes a TFA email target returned by Gigya.
type EmailRecord struct {
	ID    string `json:"id"`
	Plain string `json:"plain"`
}

// Identity manages Renault Gigya login state.
type Identity struct {
	*request.Helper
	gigya    keys.ConfigServer
	region   string
	Token    string
	PersonID string
}

// NewIdentity creates a Gigya identity client.
func NewIdentity(log *util.Logger, gigya keys.ConfigServer, region string) *Identity {
	return &Identity{
		Helper: request.NewHelper(log),
		gigya:  gigya,
		region: region,
	}
}

// Subject returns the stable provider auth id for a Renault account.
func Subject(user, region string) string {
	h := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(user)) + ":" + strings.ToLower(strings.TrimSpace(region))))
	return "renault-" + hex.EncodeToString(h[:])[:12]
}

// VerificationCodeMessage returns the UI prompt for Renault TFA.
func VerificationCodeMessage() string {
	return verificationCodeMessage
}

// SettingsKey returns the persisted trusted device key for a Renault account.
func SettingsKey(user, region string) string {
	return Subject(user, region) + "-gmid"
}

// StoredGMID returns the trusted Gigya member id for a Renault account.
func StoredGMID(user, region string) string {
	gmid, _ := settings.String(SettingsKey(user, region))
	return gmid
}

// StoreGMID stores the trusted Gigya member id for a Renault account.
func StoreGMID(user, region, gmid string) {
	settings.SetString(SettingsKey(user, region), gmid)
}

// DeleteGMID removes the trusted Gigya member id for a Renault account.
func DeleteGMID(user, region string) error {
	return settings.Delete(SettingsKey(user, region))
}

// IsTFARequired returns whether the Gigya error code means email TFA is required.
func IsTFARequired(code int) bool {
	return code == errTFAPending || code == errAccountPendingTFA
}

// TFARequired returns whether the Gigya response asks for email TFA.
func (r Response) TFARequired() bool {
	return IsTFARequired(r.ErrorCode) ||
		strings.Contains(strings.ToLower(r.ErrorMessage), "tfa") ||
		strings.Contains(strings.ToLower(r.ErrorDetails), "tfa")
}

func (v *Identity) Login(user, password string) error {
	sessionCookie, err := v.sessionCookie(user, password)

	if err == nil {
		v.Token, err = v.jwtToken(sessionCookie)
	}

	if err == nil {
		if v.PersonID, err = v.personID(sessionCookie); v.PersonID == "" {
			err = errors.New("missing personID")
		}
	}

	return err
}

func (v *Identity) sessionCookie(user, password string) (string, error) {
	res, err := v.LoginRaw(user, password, StoredGMID(user, v.region))
	if err == nil && res.ErrorCode > 0 {
		err = res.Error()
		if res.TFARequired() {
			err = api.LoginRequiredError(Subject(user, v.region))
		}
	}

	return res.SessionInfo.CookieValue, err
}

// LoginRaw performs the Gigya login call and returns the unprocessed response.
func (v *Identity) LoginRaw(user, password, gmid string) (Response, error) {
	data := url.Values{
		"loginID":  {user},
		"password": {password},
		"ApiKey":   {v.gigya.APIKey},
	}

	var res Response
	req, err := request.New(http.MethodPost, v.gigya.Target+"/accounts.login", strings.NewReader(data.Encode()), request.URLEncoding)
	if err == nil && gmid != "" {
		req.Header.Set("Cookie", "gmid="+gmid)
	}
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}

func (r Response) Error() error {
	if r.ErrorMessage != "" {
		if r.ErrorDetails != "" {
			return fmt.Errorf("%s: %s", r.ErrorMessage, r.ErrorDetails)
		}
		return errors.New(r.ErrorMessage)
	}
	if r.ErrorCode > 0 {
		return fmt.Errorf("gigya error %d", r.ErrorCode)
	}
	return nil
}

// InitTFA starts the Gigya email verification flow.
func (v *Identity) InitTFA(regToken, gmid string) (Response, error) {
	data := url.Values{
		"ApiKey":   {v.gigya.APIKey},
		"regToken": {regToken},
		"provider": {providerGigyaEmail},
		"mode":     {"verify"},
		"lang":     {"en"},
	}

	var res Response
	req, err := request.New(http.MethodPost, v.gigya.Target+"/accounts.tfa.initTFA", strings.NewReader(data.Encode()), request.URLEncoding)
	if err == nil {
		req.Header.Set("Cookie", "gmid="+gmid)
		err = v.DoJSON(req, &res)
	}

	return res, err
}

// TFAEmails returns the email targets available for TFA.
func (v *Identity) TFAEmails(gigyaAssertion, gmid string) (Response, error) {
	data := url.Values{
		"ApiKey":         {v.gigya.APIKey},
		"gigyaAssertion": {gigyaAssertion},
	}

	var res Response
	req, err := request.New(http.MethodPost, v.gigya.Target+"/accounts.tfa.email.getEmails", strings.NewReader(data.Encode()), request.URLEncoding)
	if err == nil {
		req.Header.Set("Cookie", "gmid="+gmid)
		err = v.DoJSON(req, &res)
	}

	return res, err
}

// SendVerificationCode sends a Gigya email verification code.
func (v *Identity) SendVerificationCode(gigyaAssertion, gmid, loginID string) (Response, error) {
	data := url.Values{
		"ApiKey":         {v.gigya.APIKey},
		"gigyaAssertion": {gigyaAssertion},
		"lang":           {"en"},
	}

	if emailID, err := v.emailID(gigyaAssertion, gmid, loginID); err == nil && emailID != "" {
		data.Set("emailID", emailID)
	} else {
		data.Set("email", loginID)
	}

	var res Response
	req, err := request.New(http.MethodPost, v.gigya.Target+"/accounts.tfa.email.sendVerificationCode", strings.NewReader(data.Encode()), request.URLEncoding)
	if err == nil {
		req.Header.Set("Cookie", "gmid="+gmid)
		err = v.DoJSON(req, &res)
	}

	return res, err
}

func (v *Identity) emailID(gigyaAssertion, gmid, loginID string) (string, error) {
	res, err := v.TFAEmails(gigyaAssertion, gmid)
	if err != nil || res.ErrorCode != 0 {
		return "", err
	}

	loginID = strings.ToLower(strings.TrimSpace(loginID))
	for _, email := range res.Emails {
		if strings.ToLower(strings.TrimSpace(email.Plain)) == loginID {
			return email.ID, nil
		}
	}

	return "", nil
}

// CompleteVerification verifies a Gigya email verification code.
func (v *Identity) CompleteVerification(gigyaAssertion, phvToken, gmid, code string) (Response, error) {
	data := url.Values{
		"ApiKey":         {v.gigya.APIKey},
		"gigyaAssertion": {gigyaAssertion},
		"phvToken":       {phvToken},
		"code":           {strings.TrimSpace(code)},
	}

	var res Response
	req, err := request.New(http.MethodPost, v.gigya.Target+"/accounts.tfa.email.completeVerification", strings.NewReader(data.Encode()), request.URLEncoding)
	if err == nil {
		req.Header.Set("Cookie", "gmid="+gmid)
		err = v.DoJSON(req, &res)
	}

	return res, err
}

// FinalizeTFA marks the Gigya member id as trusted.
func (v *Identity) FinalizeTFA(regToken, gigyaAssertion, providerAssertion, gmid string) (Response, error) {
	data := url.Values{
		"ApiKey":            {v.gigya.APIKey},
		"regToken":          {regToken},
		"gigyaAssertion":    {gigyaAssertion},
		"providerAssertion": {providerAssertion},
		"tempDevice":        {"false"},
	}

	var res Response
	req, err := request.New(http.MethodPost, v.gigya.Target+"/accounts.tfa.finalizeTFA", strings.NewReader(data.Encode()), request.URLEncoding)
	if err == nil {
		req.Header.Set("Cookie", "gmid="+gmid)
		err = v.DoJSON(req, &res)
	}

	return res, err
}

func (v *Identity) personID(sessionCookie string) (string, error) {
	data := url.Values{
		"apiKey":      {v.gigya.APIKey},
		"login_token": {sessionCookie},
	}

	var res Response
	uri := fmt.Sprintf("%s/accounts.getAccountInfo?%s", v.gigya.Target, data.Encode())
	err := v.GetJSON(uri, &res)

	return res.Data.PersonID, err
}

func (v *Identity) jwtToken(sessionCookie string) (string, error) {
	data := url.Values{
		"apiKey":      {v.gigya.APIKey},
		"login_token": {sessionCookie},
		"fields":      {"data.personId,data.DataCenter"},
		"expiration":  {"900"},
	}

	var res Response
	uri := fmt.Sprintf("%s/accounts.getJWT?%s", v.gigya.Target, data.Encode())
	err := v.GetJSON(uri, &res)

	return res.IDToken, err
}
