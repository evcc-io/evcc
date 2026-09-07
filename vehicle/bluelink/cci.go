package bluelink

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
	"uuid"

	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// CCIConfig holds the OneApp/CCI login parameters. Only set for EU Kia/Hyundai,
// where the legacy IDPConnect authorize endpoint is WAF-blocked.
type CCIConfig struct {
	OneAppClientID       string // OneApp OAuth2 client_id (not on the WAF block list)
	OneAppRedirectURI    string
	APIURL               string // e.g. https://cci-api-eu.kia.com
	PackageID            string // "client-id" header value (mobile app bundle id)
	ClientName           string // "client-name" header value
	OSVersion            string // "client-os-version" header value
	NotificationProvider string // "client-notification-provider-type" header value
}

const (
	cciClientVersion   = "1.3.3"
	cciMobileUserAgent = "Mozilla/5.0 (Linux; Android 4.1.1; Galaxy Nexus Build/JRO03C) AppleWebKit/535.19 (KHTML, like Gecko) Chrome/18.0.1025.166 Mobile Safari/535.19_CCS_APP_AOS"
	cciSettingsKeyFmt  = "bluelink-cci.%s.%s"
)

// cciBundle is the CCI/CCS token state. AccessToken is the CCS token used on
// the legacy ccapi endpoints, the remaining fields are required to refresh it.
type cciBundle struct {
	AccessToken              string    `json:"access_token"`  // CCS token (no "Bearer " prefix)
	RefreshToken             string    `json:"refresh_token"` // CCI refresh token
	Expiry                   time.Time `json:"expiry"`        // CCS token expiry
	DeviceID                 string    `json:"device_id"`     // client-device-id used for all CCI calls
	CCIAccessToken           string    `json:"cci_access_token"`
	ExchangeableToken        string    `json:"exchangeable_token"`
	ExchangeableRefreshToken string    `json:"exchangeable_refresh_token"`
	NonCcsToken              string    `json:"non_ccs_token"`
	NonCcsRefreshToken       string    `json:"non_ccs_refresh_token"`
	IDToken                  string    `json:"id_token"`
}

func (b cciBundle) token() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  b.AccessToken,
		RefreshToken: b.RefreshToken,
		Expiry:       b.Expiry,
	}
}

// settingsKey returns the settings key the CCI token bundle is persisted under
func (v *Identity) settingsKey() string {
	return fmt.Sprintf(cciSettingsKeyFmt, v.config.Brand, v.user)
}

// loginCCI obtains a CCS access token via the OneApp/CCI flow, preferring a
// bundle persisted from an earlier login over a fresh password login
func (v *Identity) loginCCI(password string) (*oauth2.Token, error) {
	if err := settings.Json(v.settingsKey(), &v.bundle); err == nil {
		if token := v.bundle.token(); token.Valid() {
			v.log.DEBUG.Println("cci: using persisted token")
			return token, nil
		}

		if v.bundle.RefreshToken != "" {
			token, err := v.refreshCCI(nil)
			if err == nil {
				return token, nil
			}
			v.log.WARN.Printf("cci: refreshing persisted token failed: %v", err)
		}
	}

	return v.loginCCIPassword(password)
}

// loginCCIPassword performs the headless OneApp/CCI password login: authorize,
// fetch RSA cert, signin, exchange the auth code for CCI and then CCS tokens
func (v *Identity) loginCCIPassword(password string) (*oauth2.Token, error) {
	c := v.config.CCI
	deviceID := uuid.New().String()

	v.log.DEBUG.Println("cci: logging in via OneApp/CCI password login")

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	v.Client.Jar = jar
	defer func() {
		v.Client.Jar = nil
		v.Client.CheckRedirect = nil
	}()

	// the OneApp client_id is not on the WAF block list
	authURL := fmt.Sprintf(
		"%s/auth/api/v2/user/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&lang=en&state=ccsp&country=de",
		v.config.LoginFormHost, c.OneAppClientID, c.OneAppRedirectURI,
	)
	authReq, err := request.New(http.MethodGet, authURL, nil, map[string]string{
		"User-Agent": cciMobileUserAgent,
	})
	if err != nil {
		return nil, err
	}

	authResp, err := v.Client.Do(authReq)
	if err != nil {
		return nil, err
	}
	authBody, _ := io.ReadAll(authResp.Body)
	authResp.Body.Close()

	if strings.Contains(strings.ToLower(string(authBody)), "abusing") ||
		strings.Contains(authResp.Request.URL.String(), "/error?status=400") {
		return nil, errors.New("authorize rejected as 'abusing request' — server-side WAF block, not a credentials problem")
	}

	if authResp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("authorize failed: HTTP %d (%s)", authResp.StatusCode, request.Truncate(string(authBody)))
	}

	// rsa public key used to encrypt the password for signin
	certReq, err := request.New(http.MethodGet, v.config.LoginFormHost+"/auth/api/v1/accounts/certs", nil, map[string]string{
		"User-Agent": cciMobileUserAgent,
		"Accept":     request.JSONContent,
	})
	if err != nil {
		return nil, err
	}

	var certRes struct {
		RetValue struct {
			Kid string
			N   string
			E   string
		}
	}
	if err := v.DoJSON(certReq, &certRes); err != nil {
		return nil, fmt.Errorf("fetching rsa certs failed: %w", err)
	}

	encryptedPassword, err := encryptCCIPassword(certRes.RetValue.N, certRes.RetValue.E, password)
	if err != nil {
		return nil, fmt.Errorf("encrypting password failed: %w", err)
	}

	data := url.Values{
		"client_id":             {c.OneAppClientID},
		"encryptedPassword":     {"true"},
		"password":              {encryptedPassword},
		"redirect_uri":          {c.OneAppRedirectURI},
		"scope":                 {""},
		"nonce":                 {""},
		"state":                 {"ccsp"},
		"username":              {v.user},
		"connector_session_key": {""},
		"kid":                   {certRes.RetValue.Kid},
		"_csrf":                 {""},
	}
	signinReq, err := request.New(http.MethodPost, v.config.LoginFormHost+"/auth/account/signin", strings.NewReader(data.Encode()), map[string]string{
		"Content-Type": request.FormContent,
		"User-Agent":   cciMobileUserAgent,
	})
	if err != nil {
		return nil, err
	}

	// the auth code arrives in the Location header, so redirects must not be followed
	v.Client.CheckRedirect = request.DontFollow
	signinResp, err := v.Client.Do(signinReq)
	v.Client.CheckRedirect = nil
	if err != nil {
		return nil, err
	}
	signinBody, _ := io.ReadAll(signinResp.Body)
	signinResp.Body.Close()

	if signinResp.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("signin failed: HTTP %d (%s)", signinResp.StatusCode, request.Truncate(string(signinBody)))
	}

	loc, err := signinResp.Location()
	if err != nil {
		return nil, fmt.Errorf("signin returned no valid redirect: %w", err)
	}

	code := loc.Query().Get("code")
	if code == "" {
		switch {
		case strings.Contains(loc.Path, "/web/v1/user/authorization"):
			return nil, errors.New("account consent required: log in via the manufacturer app once to accept the terms, then retry")
		case loc.Query().Get("error_description") != "":
			return nil, fmt.Errorf("signin rejected: %s", loc.Query().Get("error_description"))
		default:
			return nil, fmt.Errorf("unexpected redirect after signin: %s", request.Truncate(loc.String()))
		}
	}

	bundle, err := v.exchangeCCIToken(deviceID, code)
	if err != nil {
		return nil, err
	}
	bundle.DeviceID = deviceID

	bundle.AccessToken, bundle.Expiry, err = v.exchangeCCSToken(deviceID, bundle.CCIAccessToken, bundle.NonCcsToken, bundle.ExchangeableToken)
	if err != nil {
		return nil, err
	}

	v.persistBundle(bundle)
	v.log.DEBUG.Println("cci: login successful")

	return bundle.token(), nil
}

// refreshCCI refreshes the CCI token set and re-exchanges the CCS token. The
// token set lives on the Identity, so the passed oauth2 token is ignored.
func (v *Identity) refreshCCI(_ *oauth2.Token) (*oauth2.Token, error) {
	v.log.DEBUG.Println("cci: refreshing token")

	bundle := v.bundle
	headers := v.cciHeaders(bundle.DeviceID, bundle.CCIAccessToken, bundle.NonCcsToken, bundle.ExchangeableToken, request.JSONContent)

	body := map[string]string{
		"accessToken":              bundle.CCIAccessToken,
		"refreshToken":             bundle.RefreshToken,
		"exchangeableAccessToken":  bundle.ExchangeableToken,
		"exchangeableRefreshToken": bundle.ExchangeableRefreshToken,
		"nonCcsToken":              bundle.NonCcsToken,
		"nonCcsRefreshToken":       bundle.NonCcsRefreshToken,
		"idToken":                  bundle.IDToken,
	}

	uri := v.config.CCI.APIURL + "/domain/api/v2/auth/token-refresh"
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(body), headers)
	if err != nil {
		return nil, err
	}

	var res cciTokenResponse
	if err := v.DoJSON(req, &res); err != nil {
		return nil, fmt.Errorf("cci token refresh failed: %w", err)
	}

	res.apply(&bundle)

	bundle.AccessToken, bundle.Expiry, err = v.exchangeCCSToken(bundle.DeviceID, bundle.CCIAccessToken, bundle.NonCcsToken, bundle.ExchangeableToken)
	if err != nil {
		return nil, err
	}

	v.persistBundle(bundle)
	v.log.DEBUG.Println("cci: refresh successful")

	return bundle.token(), nil
}

func (v *Identity) persistBundle(bundle cciBundle) {
	v.bundle = bundle
	if err := settings.SetJson(v.settingsKey(), bundle); err != nil {
		v.log.WARN.Printf("cci: persisting token failed: %v", err)
	}
}

// cciTokenResponse is the response of the CCI token and token-refresh endpoints
type cciTokenResponse struct {
	AccessToken              string `json:"accessToken"`
	RefreshToken             string `json:"refreshToken"`
	NonCcsToken              string `json:"nonCcsToken"`
	ExchangeableAccessToken  string `json:"exchangeableAccessToken"`
	ExchangeableRefreshToken string `json:"exchangeableRefreshToken"`
	NonCcsRefreshToken       string `json:"nonCcsRefreshToken"`
	IDToken                  string `json:"idToken"`
}

// apply copies the non-empty response fields into bundle, keeping unchanged ones
func (res cciTokenResponse) apply(bundle *cciBundle) {
	for _, f := range []struct {
		val string
		dst *string
	}{
		{res.AccessToken, &bundle.CCIAccessToken},
		{res.RefreshToken, &bundle.RefreshToken},
		{res.NonCcsToken, &bundle.NonCcsToken},
		{res.ExchangeableAccessToken, &bundle.ExchangeableToken},
		{res.ExchangeableRefreshToken, &bundle.ExchangeableRefreshToken},
		{res.NonCcsRefreshToken, &bundle.NonCcsRefreshToken},
		{res.IDToken, &bundle.IDToken},
	} {
		if f.val != "" {
			*f.dst = f.val
		}
	}
}

// exchangeCCIToken exchanges an authorization code for the CCI token set
func (v *Identity) exchangeCCIToken(deviceID, code string) (cciBundle, error) {
	uri := v.config.CCI.APIURL + "/domain/api/v1/auth/token?code=" + url.QueryEscape(code)
	req, err := request.New(http.MethodPost, uri, nil, v.cciHeaders(deviceID, "", "", "", ""))
	if err != nil {
		return cciBundle{}, err
	}

	var res cciTokenResponse
	if err := v.DoJSON(req, &res); err != nil {
		return cciBundle{}, fmt.Errorf("cci token exchange failed: %w", err)
	}

	var bundle cciBundle
	res.apply(&bundle)

	return bundle, nil
}

// exchangeCCSToken exchanges a CCI access token for a CCS token, which the
// legacy ccapi vehicle/control endpoints accept as Bearer access_token
func (v *Identity) exchangeCCSToken(deviceID, cciAccessToken, nonCcsToken, exchangeableToken string) (string, time.Time, error) {
	uri := v.config.CCI.APIURL + "/domain/api/v1/auth/token-exchange?serviceType=CCS"
	headers := v.cciHeaders(deviceID, cciAccessToken, nonCcsToken, exchangeableToken, "")

	req, err := request.New(http.MethodPost, uri, nil, headers)
	if err != nil {
		return "", time.Time{}, err
	}

	var res struct {
		AccessToken string `json:"accessToken"`
		ExpiresTime int64  `json:"expiresTime"` // unix seconds
	}
	if err := v.DoJSON(req, &res); err != nil {
		return "", time.Time{}, fmt.Errorf("ccs token exchange failed: %w", err)
	}

	if res.AccessToken == "" {
		return "", time.Time{}, errors.New("ccs token exchange returned no access token")
	}

	return res.AccessToken, parseCCSExpiry(res.ExpiresTime), nil
}

const (
	ccsExpiryFallback    = time.Hour      // used when expiresTime is missing or implausible
	ccsExpiryMaxValidity = 24 * time.Hour // upper bound of a plausible expiry
)

// parseCCSExpiry interprets the token-exchange expiresTime, falling back to a
// fixed lifetime rather than risking an always-expired token
func parseCCSExpiry(expiresTime int64) time.Time {
	now := time.Now()

	if t := time.Unix(expiresTime, 0); t.After(now) && t.Before(now.Add(ccsExpiryMaxValidity)) {
		return t
	}

	return now.Add(ccsExpiryFallback)
}

// cciHeaders builds the headers required by the CCI API. The token parameters
// are empty before the initial code exchange, contentType for bodyless requests
func (v *Identity) cciHeaders(deviceID, cciAccessToken, nonCcsToken, exchangeableToken, contentType string) map[string]string {
	c := v.config.CCI

	headers := map[string]string{
		"client-id":                         c.PackageID,
		"client-name":                       c.ClientName,
		"client-version":                    cciClientVersion,
		"client-os-code":                    "ios",
		"client-os-version":                 c.OSVersion,
		"client-device-id":                  deviceID,
		"client-device-model":               "iPhone",
		"client-notification-provider-type": c.NotificationProvider,
		"locale":                            strings.ToUpper(v.language),
		"timezone":                          time.Now().Format("-07:00"),
		"Accept":                            request.JSONContent,
		"Accept-Language":                   v.language,
		"User-Agent":                        cciMobileUserAgent,
	}

	if nonCcsToken != "" {
		headers["Authentication"] = nonCcsToken
	}
	if cciAccessToken != "" {
		headers["authorization"] = "Bearer " + strings.TrimPrefix(strings.TrimSpace(cciAccessToken), "Bearer ")
	}
	if exchangeableToken != "" {
		headers["exchangeable-token"] = exchangeableToken
		headers["non-ccs-token"] = nonCcsToken
	}

	if contentType != "" {
		headers["Content-Type"] = contentType
	}

	return headers
}

// encryptCCIPassword RSA/PKCS1v15-encrypts password using the JWK public key
// returned by the /accounts/certs endpoint and hex-encodes the ciphertext
func encryptCCIPassword(nb64, eb64, password string) (string, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(nb64, "="))
	if err != nil {
		return "", fmt.Errorf("invalid rsa modulus: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(eb64, "="))
	if err != nil {
		return "", fmt.Errorf("invalid rsa exponent: %w", err)
	}

	e := 0
	for _, b := range eBytes {
		e = e<<8 | int(b)
	}

	pub := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: e,
	}

	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, pub, []byte(password))
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(ciphertext), nil
}
