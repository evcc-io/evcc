package bluelink

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/request"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// CCIConfig holds the OneApp/CCI login parameters for brands where the
// legacy IDPConnect authorize endpoint is blocked by Hyundai's WAF (as of
// 2026-08, see https://github.com/Hyundai-Kia-Connect/hyundai_kia_connect_api/issues/1273).
// It is only set for EU Kia/Hyundai — Genesis EU and Hyundai AU are not
// WAF-affected and keep using the legacy authorize/refresh flow unchanged
// (Config.CCI == nil for those brands).
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
	cciClientVersion    = "1.3.3"
	cciMobileUserAgent  = "Mozilla/5.0 (Linux; Android 4.1.1; Galaxy Nexus Build/JRO03C) AppleWebKit/535.19 (KHTML, like Gecko) Chrome/18.0.1025.166 Mobile Safari/535.19_CCS_APP_AOS"
	cciBundlePrefix     = "cci1:" // marks a compound CCI token bundle in the config `password` field
	cciSettingsKeyFmt   = "bluelink-cci.%s.%s"
	cciTimezoneLocation = "Europe/Berlin"
)

// cciBundle is the full CCI/CCS token state needed to authenticate against the
// legacy ccapi:8080 vehicle/control endpoints (AccessToken) and to refresh
// without repeating the full password login (all other fields). It is used
// both as the persistence format (server/db/settings, keyed per brand+user)
// and as the wire format for a compound bundle string that a user can paste
// directly into the `password` config field (see decodeCCIBundle).
type cciBundle struct {
	AccessToken              string    `json:"access_token"`  // CCS token (no "Bearer " prefix) — used for the legacy vehicle API
	RefreshToken             string    `json:"refresh_token"` // CCI refresh token
	Expiry                   time.Time `json:"expiry"`        // CCS token expiry
	DeviceID                 string    `json:"device_id"`     // stable client-device-id used for all CCI calls of this session
	CCIAccessToken           string    `json:"cci_access_token"`
	ExchangeableToken        string    `json:"exchangeable_token"`
	ExchangeableRefreshToken string    `json:"exchangeable_refresh_token"`
	NonCcsToken              string    `json:"non_ccs_token"`
	NonCcsRefreshToken       string    `json:"non_ccs_refresh_token"`
	IDToken                  string    `json:"id_token"`
}

// extra keys used to carry the cciBundle fields through oauth2.Token.Extra,
// so the existing oauth.RefreshTokenSource (which only knows about
// AccessToken/RefreshToken/Expiry) can be reused unchanged for the CCI flow.
const (
	extraDeviceID                 = "device_id"
	extraCCIAccessToken           = "cci_access_token"
	extraExchangeableToken        = "exchangeable_token"
	extraExchangeableRefreshToken = "exchangeable_refresh_token"
	extraNonCcsToken              = "non_ccs_token"
	extraNonCcsRefreshToken       = "non_ccs_refresh_token"
	extraIDToken                  = "id_token"
)

// token converts the bundle into an oauth2.Token, carrying the CCI-specific
// fields in Extra so they survive a round-trip through oauth.RefreshTokenSource.
func (b cciBundle) token() *oauth2.Token {
	t := &oauth2.Token{
		AccessToken:  b.AccessToken,
		RefreshToken: b.RefreshToken,
		Expiry:       b.Expiry,
	}

	return t.WithExtra(map[string]any{
		extraDeviceID:                 b.DeviceID,
		extraCCIAccessToken:           b.CCIAccessToken,
		extraExchangeableToken:        b.ExchangeableToken,
		extraExchangeableRefreshToken: b.ExchangeableRefreshToken,
		extraNonCcsToken:              b.NonCcsToken,
		extraNonCcsRefreshToken:       b.NonCcsRefreshToken,
		extraIDToken:                  b.IDToken,
	})
}

func cciExtraString(token *oauth2.Token, key string) string {
	s, _ := token.Extra(key).(string)
	return s
}

// bundleFromToken is the inverse of cciBundle.token().
func bundleFromToken(token *oauth2.Token) cciBundle {
	return cciBundle{
		AccessToken:              token.AccessToken,
		RefreshToken:             token.RefreshToken,
		Expiry:                   token.Expiry,
		DeviceID:                 cciExtraString(token, extraDeviceID),
		CCIAccessToken:           cciExtraString(token, extraCCIAccessToken),
		ExchangeableToken:        cciExtraString(token, extraExchangeableToken),
		ExchangeableRefreshToken: cciExtraString(token, extraExchangeableRefreshToken),
		NonCcsToken:              cciExtraString(token, extraNonCcsToken),
		NonCcsRefreshToken:       cciExtraString(token, extraNonCcsRefreshToken),
		IDToken:                  cciExtraString(token, extraIDToken),
	}
}

// decodeCCIBundle decodes a compound CCI token bundle from the config
// `password` field. Wire format: "cci1:" + base64url(json(cciBundle)) with no
// padding. This lets a user who has already obtained a CCI token set with an
// external tool configure evcc without ever storing their live account
// password, mirroring how the pre-existing legacy refresh_token worked.
func decodeCCIBundle(s string) (cciBundle, bool) {
	if !strings.HasPrefix(s, cciBundlePrefix) {
		return cciBundle{}, false
	}

	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(s, cciBundlePrefix))
	if err != nil {
		return cciBundle{}, false
	}

	var b cciBundle
	if err := json.Unmarshal(raw, &b); err != nil {
		return cciBundle{}, false
	}

	return b, true
}

// legacyRefreshTokenPattern matches the shape evcc has always accepted here:
// an opaque legacy IDPConnect refresh_token, historically 48 uppercase
// letters/digits (hyundai_kia_connect_api uses the exact `^[A-Z0-9]{48}$`);
// widened slightly to 40-60 chars to tolerate minor length variance across
// brands/regions. Note this is a *character-class* check, not just a length
// check: a real account password virtually always contains lowercase letters
// and/or symbols, which this pattern rejects, so it is never misclassified
// as a legacy token merely for being short.
var legacyRefreshTokenPattern = regexp.MustCompile(`^[A-Z0-9]{40,60}$`)

// looksLikeLegacyRefreshToken reports whether password has the shape of a
// legacy IDPConnect refresh_token (see legacyRefreshTokenPattern). This keeps
// the behaviour for every user who still holds a valid pre-block legacy
// refresh_token completely unchanged, while routing anything else (a real
// account password, or a cci1: bundle) to the CCI-capable path.
func looksLikeLegacyRefreshToken(password string) bool {
	return legacyRefreshTokenPattern.MatchString(password)
}

// settingsKey returns the server/db/settings key under which the CCI token
// bundle for this identity is persisted, mirroring the pattern used by e.g.
// vehicle/tesla and vehicle/mercedes for their OAuth2 tokens.
func (v *Identity) settingsKey() string {
	return fmt.Sprintf(cciSettingsKeyFmt, v.config.Brand, v.user)
}

// loginCCI obtains a usable CCS access token for EU Kia/Hyundai via the
// OneApp/CCI flow. Priority order:
//  1. a token bundle persisted from a prior login/refresh (server/db/settings) —
//     avoids hitting the login endpoint (and its stricter rate limiting/WAF)
//     on every evcc restart.
//  2. a CCI token bundle supplied directly in the password field (see
//     decodeCCIBundle) — lets users who don't want evcc to hold their live
//     account password generate the bundle once with an external tool.
//  3. a full interactive username+password login.
func (v *Identity) loginCCI(password string) (*oauth2.Token, error) {
	var persisted cciBundle
	if err := settings.Json(v.settingsKey(), &persisted); err == nil {
		if token, err := v.tokenOrRefresh(persisted); err == nil {
			v.log.DEBUG.Println("cci: using persisted token from database")
			return token, nil
		} else {
			v.log.WARN.Printf("cci: persisted token invalid or refresh failed, falling back: %v", err)
		}
	}

	if bundle, ok := decodeCCIBundle(password); ok {
		token, err := v.tokenOrRefresh(bundle)
		if err != nil {
			return nil, fmt.Errorf("cci token bundle in config is invalid or expired: %w", err)
		}
		return token, nil
	}

	return v.loginCCIPassword(password)
}

// tokenOrRefresh returns the bundle's token directly if still valid, otherwise
// refreshes it.
func (v *Identity) tokenOrRefresh(bundle cciBundle) (*oauth2.Token, error) {
	token := bundle.token()
	if token.Valid() {
		return token, nil
	}
	if token.RefreshToken == "" {
		return nil, errors.New("no refresh token available")
	}
	return v.refreshCCI(token)
}

// loginCCIPassword performs the OneApp/CCI headless password login (EU
// Kia/Hyundai). It bypasses the legacy IDPConnect authorize endpoint, which
// is WAF-blocked for the legacy client_id. Steps:
//  1. GET  {LoginFormHost}/auth/api/v2/user/oauth2/authorize (OneApp client_id — not WAF-blocked)
//  2. GET  {LoginFormHost}/auth/api/v1/accounts/certs        (RSA public key for password encryption)
//  3. POST {LoginFormHost}/auth/account/signin                (RSA-encrypted password, redirect not followed)
//  4. POST {CCI.APIURL}/domain/api/v1/auth/token              (auth code -> CCI token set)
//  5. POST {CCI.APIURL}/domain/api/v1/auth/token-exchange     (CCI token -> CCS token, usable on legacy ccapi:8080)
func (v *Identity) loginCCIPassword(password string) (*oauth2.Token, error) {
	c := v.config.CCI
	deviceID := uuid.NewString()

	v.log.INFO.Println("cci: logging in via OneApp/CCI password login")

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	v.Client.Jar = jar
	defer func() {
		v.Client.Jar = nil
		v.Client.CheckRedirect = nil
	}()

	// Step 1: authorize. The OneApp client_id is not on the WAF block list, so
	// this succeeds where the legacy client_id's authorize is rejected.
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
		return nil, errors.New("login blocked: IDPConnect authorize rejected the request ('abusing request'). " +
			"This is a server-side WAF block, not a credentials problem")
	}

	// Step 2: RSA public key used to encrypt the password for signin.
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

	// Step 3: signin with the RSA-encrypted password. The auth code arrives in
	// the Location header of the 302 response, so redirects must not be followed.
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

	v.Client.CheckRedirect = request.DontFollow
	signinResp, err := v.Client.Do(signinReq)
	if err != nil {
		return nil, err
	}
	signinBody, _ := io.ReadAll(signinResp.Body)
	signinResp.Body.Close()

	if signinResp.StatusCode != http.StatusFound {
		return nil, signinFailureError(signinResp.StatusCode, string(signinBody))
	}

	location := signinResp.Header.Get("Location")
	loc, err := url.Parse(location)
	if err != nil {
		return nil, fmt.Errorf("signin returned unparseable redirect: %w", err)
	}

	code := loc.Query().Get("code")
	if code == "" {
		switch {
		case strings.Contains(strings.ToLower(location), "error"):
			desc := loc.Query().Get("error_description")
			if desc == "" {
				desc = "unknown"
			}
			if looksCredentialRelated(desc) {
				return nil, fmt.Errorf("authentication rejected: %s — check username and password", desc)
			}
			return nil, fmt.Errorf("authentication rejected: %s — this may indicate a server-side issue rather than incorrect credentials", desc)
		case strings.Contains(location, "/web/v1/user/authorization"):
			return nil, errors.New("account consent required: please log in via the manufacturer app or browser once to accept the terms, then retry")
		case strings.Contains(location, "authorize"):
			return nil, errors.New("authentication failed — returned to login page, check username and password")
		default:
			return nil, fmt.Errorf("unexpected redirect after signin: %s", truncate(location, 250))
		}
	}

	// Step 4: exchange the authorization code for the CCI token set.
	bundle, err := v.exchangeCCIToken(deviceID, code)
	if err != nil {
		return nil, err
	}
	bundle.DeviceID = deviceID

	// Step 5: exchange the CCI access token for a CCS token, accepted by the
	// legacy ccapi:8080 vehicle/control endpoints.
	ccsToken, ccsValidUntil, err := v.exchangeCCSToken(deviceID, bundle.CCIAccessToken, bundle.NonCcsToken, bundle.ExchangeableToken)
	if err != nil {
		return nil, err
	}
	bundle.AccessToken = ccsToken
	bundle.Expiry = ccsValidUntil

	if err := settings.SetJson(v.settingsKey(), bundle); err != nil {
		v.log.WARN.Printf("cci: persisting token failed: %v", err)
	}

	v.log.INFO.Println("cci: login successful")

	return bundle.token(), nil
}

// refreshCCI refreshes the CCI token set via cci-api-eu/.../v2/auth/token-refresh
// (the full CCI token set is required in the request body), then re-exchanges
// the CCS token. Called automatically by oauth.RefreshTokenSource whenever the
// current CCS token has expired.
func (v *Identity) refreshCCI(token *oauth2.Token) (*oauth2.Token, error) {
	v.log.DEBUG.Println("cci: refreshing token")

	bundle := bundleFromToken(token)

	c := v.config.CCI
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

	uri := c.APIURL + "/domain/api/v2/auth/token-refresh"
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(body), headers)
	if err != nil {
		return nil, err
	}

	var res struct {
		AccessToken              string `json:"accessToken"`
		RefreshToken             string `json:"refreshToken"`
		NonCcsToken              string `json:"nonCcsToken"`
		ExchangeableAccessToken  string `json:"exchangeableAccessToken"`
		ExchangeableRefreshToken string `json:"exchangeableRefreshToken"`
		NonCcsRefreshToken       string `json:"nonCcsRefreshToken"`
		IDToken                  string `json:"idToken"`
	}
	if err := v.DoJSON(req, &res); err != nil {
		return nil, fmt.Errorf("cci token refresh failed: %w", err)
	}

	if res.AccessToken != "" {
		bundle.CCIAccessToken = res.AccessToken
	}
	if res.RefreshToken != "" {
		bundle.RefreshToken = res.RefreshToken
	}
	if res.NonCcsToken != "" {
		bundle.NonCcsToken = res.NonCcsToken
	}
	if res.ExchangeableAccessToken != "" {
		bundle.ExchangeableToken = res.ExchangeableAccessToken
	}
	if res.ExchangeableRefreshToken != "" {
		bundle.ExchangeableRefreshToken = res.ExchangeableRefreshToken
	}
	if res.NonCcsRefreshToken != "" {
		bundle.NonCcsRefreshToken = res.NonCcsRefreshToken
	}
	if res.IDToken != "" {
		bundle.IDToken = res.IDToken
	}

	// Note: the upstream reference implementation additionally inspects a
	// Set-Cookie: t=<token> response header for an updated exchangeable
	// token. That is a supplementary/optional signal (the JSON body above is
	// the primary source) and is intentionally not replicated here to keep
	// this port simple; revisit if refreshes are observed to go stale despite
	// a 200 response.

	ccsToken, ccsValidUntil, err := v.exchangeCCSToken(bundle.DeviceID, bundle.CCIAccessToken, bundle.NonCcsToken, bundle.ExchangeableToken)
	if err != nil {
		return nil, err
	}
	bundle.AccessToken = ccsToken
	bundle.Expiry = ccsValidUntil

	if err := settings.SetJson(v.settingsKey(), bundle); err != nil {
		v.log.WARN.Printf("cci: persisting refreshed token failed: %v", err)
	}

	v.log.DEBUG.Println("cci: refresh successful")

	return bundle.token(), nil
}

// exchangeCCIToken exchanges an authorization code for the CCI token set
// (step 4 of the login flow).
func (v *Identity) exchangeCCIToken(deviceID, code string) (cciBundle, error) {
	c := v.config.CCI

	uri := c.APIURL + "/domain/api/v1/auth/token?code=" + url.QueryEscape(code)
	headers := v.cciHeaders(deviceID, "", "", "", "")

	req, err := request.New(http.MethodPost, uri, nil, headers)
	if err != nil {
		return cciBundle{}, err
	}

	var res struct {
		AccessToken              string `json:"accessToken"`
		RefreshToken             string `json:"refreshToken"`
		NonCcsToken              string `json:"nonCcsToken"`
		ExchangeableAccessToken  string `json:"exchangeableAccessToken"`
		ExchangeableRefreshToken string `json:"exchangeableRefreshToken"`
		NonCcsRefreshToken       string `json:"nonCcsRefreshToken"`
		IDToken                  string `json:"idToken"`
	}
	if err := v.DoJSON(req, &res); err != nil {
		return cciBundle{}, fmt.Errorf("cci token exchange failed: %w", err)
	}

	return cciBundle{
		RefreshToken:             res.RefreshToken,
		CCIAccessToken:           res.AccessToken,
		ExchangeableToken:        res.ExchangeableAccessToken,
		ExchangeableRefreshToken: res.ExchangeableRefreshToken,
		NonCcsToken:              res.NonCcsToken,
		NonCcsRefreshToken:       res.NonCcsRefreshToken,
		IDToken:                  res.IDToken,
	}, nil
}

// exchangeCCSToken exchanges a CCI access token for a CCS token (step 5 of
// the login flow, and the final step of every refresh). The CCS token is
// accepted by the legacy ccapi:8080 vehicle/control endpoints as the Bearer
// access_token, unchanged from how evcc has always used it.
func (v *Identity) exchangeCCSToken(deviceID, cciAccessToken, nonCcsToken, exchangeableToken string) (string, time.Time, error) {
	c := v.config.CCI

	uri := c.APIURL + "/domain/api/v1/auth/token-exchange?serviceType=CCS"
	headers := v.cciHeaders(deviceID, cciAccessToken, nonCcsToken, exchangeableToken, "")

	req, err := request.New(http.MethodPost, uri, nil, headers)
	if err != nil {
		return "", time.Time{}, err
	}

	var res struct {
		AccessToken    string `json:"accessToken"`
		CcsAccessToken string `json:"ccsAccessToken"`
		ExpiresTime    int64  `json:"expiresTime"` // epoch, unit unconfirmed - see parseCCSExpiry
	}
	if err := v.DoJSON(req, &res); err != nil {
		return "", time.Time{}, fmt.Errorf("ccs token exchange failed: %w", err)
	}

	token := res.AccessToken
	if token == "" {
		token = res.CcsAccessToken
	}
	if token == "" {
		return "", time.Time{}, errors.New("ccs token exchange returned no access token")
	}

	return token, parseCCSExpiry(res.ExpiresTime), nil
}

// ccsExpiryFallback is used whenever expiresTime is absent or doesn't parse to
// a plausible near-future timestamp.
const ccsExpiryFallback = time.Hour

// ccsExpiryPlausibleWindow bounds what counts as a "plausible" expiry: far
// enough in the future to not immediately trigger a refresh, but not
// absurdly far out (which would indicate a unit/parsing error rather than a
// genuinely long-lived token).
const ccsExpiryPlausibleWindow = 24 * time.Hour

// isPlausibleCCSExpiry reports whether t is far enough in the future to not
// immediately trigger a refresh, but not absurdly far out (which would
// indicate a unit/parsing error rather than a genuinely long-lived token).
// Kept separate from parseCCSExpiry so "is this plausible" (policy) stays
// decoupled from "which unit is this" (unit detection).
func isPlausibleCCSExpiry(now, t time.Time) bool {
	return t.After(now) && t.Before(now.Add(ccsExpiryPlausibleWindow))
}

// parseCCSExpiry interprets the token-exchange response's expiresTime field.
// The upstream hyundai_kia_connect_api reference treats it as a millisecond
// epoch, but a live test against the real Kia backend showed a token
// considered immediately expired right after a fresh login - i.e. every
// subsequent API call triggered a full refresh round-trip instead of roughly
// one per hour. That symptom is consistent with expiresTime actually being a
// *second* epoch (interpreting ~10-digit second values as milliseconds lands
// in January 1970). To be robust either way - and to never again let a
// units mismatch silently produce an always-expired token - this tries the
// millisecond interpretation first, falls back to seconds, and if neither
// lands in a plausible near-future window, ignores the field entirely and
// uses a safe fixed default instead of risking a refresh storm.
func parseCCSExpiry(expiresTime int64) time.Time {
	if expiresTime <= 0 {
		return time.Now().Add(ccsExpiryFallback)
	}

	now := time.Now()

	if ms := time.UnixMilli(expiresTime); isPlausibleCCSExpiry(now, ms) {
		return ms
	}
	if sec := time.Unix(expiresTime, 0); isPlausibleCCSExpiry(now, sec) {
		return sec
	}

	return now.Add(ccsExpiryFallback)
}

// cciBaseHeaders builds the static per-device/per-client headers required by
// the CCI API (cci-api-eu.{hyundai,kia}.com) - everything that doesn't depend
// on the current auth/exchange state.
func (v *Identity) cciBaseHeaders(deviceID string) map[string]string {
	c := v.config.CCI

	return map[string]string{
		"client-id":                         c.PackageID,
		"client-name":                       c.ClientName,
		"client-version":                    cciClientVersion,
		"client-os-code":                    "ios",
		"client-os-version":                 c.OSVersion,
		"client-device-id":                  deviceID,
		"client-device-model":               "iPhone",
		"client-notification-provider-type": c.NotificationProvider,
		"locale":                            strings.ToUpper(v.language),
		"timezone":                          cciTimezoneOffset(),
		"Accept":                            request.JSONContent,
		"Accept-Language":                   v.language,
		"User-Agent":                        cciMobileUserAgent,
	}
}

// cciAuthHeaders adds the auth-token headers to h, when available.
// cciAccessToken/nonCcsToken/exchangeableToken are optional (pass "" when not
// yet available, e.g. for the initial code exchange).
func cciAuthHeaders(h map[string]string, cciAccessToken, nonCcsToken, exchangeableToken string) {
	if nonCcsToken != "" {
		h["Authentication"] = nonCcsToken
	}
	if cciAccessToken != "" {
		h["authorization"] = "Bearer " + strings.TrimPrefix(strings.TrimSpace(cciAccessToken), "Bearer ")
	}
	if exchangeableToken != "" {
		h["exchangeable-token"] = exchangeableToken
		h["non-ccs-token"] = nonCcsToken
	}
}

// cciBodyHeaders adds the content-type/content-length header to h: a JSON
// body when contentType is given, or the bodyless requests that carry all
// state via query parameters and headers otherwise.
func cciBodyHeaders(h map[string]string, contentType string) {
	if contentType != "" {
		h["Content-Type"] = contentType
	} else {
		h["Content-Length"] = "0"
	}
}

// cciHeaders builds the full set of headers required by the CCI API,
// combining the static client headers with the auth and body headers for the
// current request.
func (v *Identity) cciHeaders(deviceID, cciAccessToken, nonCcsToken, exchangeableToken, contentType string) map[string]string {
	headers := v.cciBaseHeaders(deviceID)
	cciAuthHeaders(headers, cciAccessToken, nonCcsToken, exchangeableToken)
	cciBodyHeaders(headers, contentType)

	return headers
}

// cciTimezoneOffset returns the current UTC offset of the CCI home region
// (Central Europe) as "+HH:MM"/"-HH:MM", falling back to the local system
// timezone if the tzdata for Europe/Berlin is unavailable.
func cciTimezoneOffset() string {
	loc, err := time.LoadLocation(cciTimezoneLocation)
	if err != nil {
		loc = time.Local
	}
	return time.Now().In(loc).Format("-07:00")
}

// encryptCCIPassword RSA/PKCS1v15-encrypts password using the JWK public key
// (base64url-encoded modulus/exponent, as returned by the CCI
// /accounts/certs endpoint) and returns the ciphertext as a lowercase hex
// string, as required by the signin endpoint's encryptedPassword=true mode.
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

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// looksCredentialRelated reports whether a signin error message actually
// talks about credentials, as opposed to some other kind of failure.
func looksCredentialRelated(s string) bool {
	s = strings.ToLower(s)
	for _, kw := range []string{"password", "credential", "invalid_grant", "invalid_user", "username", "login"} {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

// signinFailureError phrases a non-302 signin response. Unlike the "302 to
// an error location" case handled by the caller (which is how a real Kia/
// Hyundai wrong-password rejection is normally surfaced), a raw non-302
// status at this stage is structurally different and more often indicates a
// transport/server-side problem - rate limiting, a WAF layer other than the
// one checked in step 1, or an API change - than incorrect credentials. Only
// blame credentials when the response actually looks like an auth rejection
// (HTTP 401, or a body that explicitly talks about credentials); otherwise
// phrase it as an unexpected response so users/operators don't waste time
// re-checking a password that was never the problem.
func signinFailureError(statusCode int, body string) error {
	body = truncate(body, 200)
	if statusCode == http.StatusUnauthorized || looksCredentialRelated(body) {
		return fmt.Errorf("signin failed: HTTP %d — check username and password (%s)", statusCode, body)
	}
	return fmt.Errorf("signin failed: unexpected HTTP %d response (%s) — this may indicate a server-side issue or API change rather than incorrect credentials", statusCode, body)
}
