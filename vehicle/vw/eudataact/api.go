package eudataact

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
	"github.com/samber/lo"
	"golang.org/x/net/publicsuffix"
)

// https://github.com/TA2k/ioBroker.vw-connect (lib/euDataAct.js)

const (
	// BaseURL is the EU Data Act portal that delivers the mandated vehicle datasets
	BaseURL = "https://eu-data-act.drivesomethinggreater.com"
	// RedirectURI is the OIDC redirect target registered for the portal
	RedirectURI = BaseURL + "/login"
	// Scope is the OIDC scope requested for the EU Data Act flow
	Scope = "openid cars profile"
)

var portalHost = strings.TrimPrefix(BaseURL, "https://")

var ErrNotConfigured = errors.New("EU Data Act subscription not configured")

// API is the EU Data Act portal client. It authenticates through the VW group
// identity service and reads vehicle data from the portal's data delivery API.
//
// Unlike the WeConnect/BFF APIs the portal is not a live telemetry service: it
// stores a dataset (a zipped JSON document) roughly every 15 minutes that the
// user has to enable once in the browser. Reading vehicle data means downloading
// the newest dataset and decoding its flat list of data points.
type API struct {
	*request.Helper
	log            *util.Logger
	brand          brand
	user, password string
	jar            sessionJar
	loginMu        sync.Mutex    // serializes logins
	loginGen       atomic.Uint64 // incremented on each successful login
}

// sessionJar lets login replace the session cookies while other vehicles use the client.
type sessionJar struct {
	atomic.Pointer[cookiejar.Jar]
}

func (j *sessionJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.Load().SetCookies(u, cookies)
}

func (j *sessionJar) Cookies(u *url.URL) []*http.Cookie {
	return j.Load().Cookies(u)
}

// apiKey identifies a portal account. All vehicles of the same brand and user
// share one authenticated client.
type apiKey struct {
	brand brand
	user  string
}

var (
	apiMu  sync.Mutex
	apiReg = make(map[apiKey]*API)
)

// NewAPI returns the EU Data Act client for the given brand and user, performing
// the initial login on first use. Subsequent calls for the same brand and user
// return the already authenticated client so that several vehicles of one
// account share a single portal session instead of competing for it.
func NewAPI(log *util.Logger, brandName, user, password string) (*API, error) {
	b, ok := resolveBrand(brandName)
	if !ok {
		return nil, fmt.Errorf("unknown brand: %s", brandName)
	}

	key := apiKey{brand: b, user: user}

	apiMu.Lock()
	defer apiMu.Unlock()

	if v, ok := apiReg[key]; ok {
		return v, nil
	}

	v := &API{
		Helper:   request.NewHelper(log),
		log:      log,
		brand:    b,
		user:     user,
		password: password,
	}
	v.Client.Jar = &v.jar

	if err := v.login(0); err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	apiReg[key] = v

	return v, nil
}

// login performs the OIDC authorization-code flow against the VW identity
// service. The portal relies on the session cookies that are set while the
// browser follows the redirect chain back to RedirectURI, so the cookie jar is
// kept on the client for all subsequent data calls.
// Logins are skipped if another caller has logged in since gen was observed.
func (v *API) login(gen uint64) error {
	v.loginMu.Lock()
	defer v.loginMu.Unlock()

	if v.loginGen.Load() != gen {
		return nil
	}

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return err
	}

	// private client copy, data requests keep their jar and redirect policy until login succeeds
	client := *v.Client
	client.Jar = jar

	identityHost := strings.TrimPrefix(vwidentity.BaseURL, "https://")
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		// Cookies from the redirect response are already stored in the jar.
		if req.URL.Scheme != "https" || isUserPage(req.URL) || isMarketingConsentPage(req.URL) {
			return http.ErrUseLastResponse
		}
		if !isLoginHost(req.URL.Host, identityHost) {
			return fmt.Errorf("unexpected login redirect host %s", req.URL.Host)
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}

	// start the OIDC authorize flow
	q := url.Values{
		"client_id":     {v.brand.clientID},
		"response_type": {"code"},
		"scope":         {Scope},
		"state":         {fmt.Sprintf("de__en__%s", v.brand.state)},
		"redirect_uri":  {RedirectURI},
		"prompt":        {"login"},
		"nonce":         {lo.RandomString(43, lo.LettersCharset)},
	}

	resp, err := client.Get(vwidentity.Config.AuthURL + "?" + q.Encode())
	if err != nil {
		return err
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	// email/identifier step
	vars, err := vwidentity.FormValues(bytes.NewReader(body), "form#emailPasswordForm")
	if err != nil {
		return fmt.Errorf("identifier form: %w (portal layout may have changed)", err)
	}

	uri := vwidentity.BaseURL + vars.Action
	resp, err = client.PostForm(uri, url.Values{
		"_csrf":      {vars.Inputs["_csrf"]},
		"relayState": {vars.Inputs["relayState"]},
		"hmac":       {vars.Inputs["hmac"]},
		"email":      {v.user},
	})
	if err != nil {
		return err
	}

	params, err := vwidentity.ParseCredentialsPage(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("credentials page: %w", err)
	}
	if params.TemplateModel.Error != "" {
		return errors.New(params.TemplateModel.Error)
	}

	// password/authenticate step - the client follows the redirect chain back to
	// the portal which sets the session cookie
	uri = strings.ReplaceAll(uri, params.TemplateModel.IdentifierUrl, params.TemplateModel.PostAction)
	resp, err = client.PostForm(uri, url.Values{
		"_csrf":      {params.CsrfToken},
		"relayState": {params.TemplateModel.RelayState},
		"hmac":       {params.TemplateModel.Hmac},
		"email":      {v.user},
		"password":   {v.password},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	final, err := loginLocation(resp)
	if err != nil {
		return err
	}

	// Skip optional marketing consent using its callback without fetching the page.
	if cb, err := vwidentity.MarketingConsentCallback(final); err != nil {
		return err
	} else if cb != nil {
		if final.Scheme != "https" || !isLoginHost(final.Host, identityHost) || !validMarketingCallback(cb, identityHost) {
			return errors.New("unexpected marketing consent callback URL")
		}
		resp, err = client.Get(cb.String())
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if final, err = loginLocation(resp); err != nil {
			return err
		}
	}

	// a successful login lands on the portal; a remaining signin/consent url means
	// the user has not completed the one-time browser consent and vehicle linking
	if strings.Contains(final.Path, "signin-service") || strings.Contains(final.Path, "/consent") || strings.Contains(final.Path, "/error") {
		return api.UrlError(
			fmt.Sprintf("login did not complete- open the portal and confirm consent: %s", final),
			final,
		)
	}
	if final.Host != portalHost {
		return fmt.Errorf("login did not complete: unexpected landing host %s", final.Host)
	}
	if !isUserPage(final) {
		return errors.New("login did not complete: unexpected landing page")
	}

	v.jar.Store(jar)
	v.loginGen.Add(1)

	return nil
}

func isLoginHost(host, identityHost string) bool {
	return strings.EqualFold(host, identityHost) || strings.EqualFold(host, portalHost)
}

func isMarketingConsentPage(u *url.URL) bool {
	return strings.EqualFold(u.Host, strings.TrimPrefix(vwidentity.BaseURL, "https://")) &&
		strings.Contains(u.Path, "/consent/marketing/")
}

func validMarketingCallback(u *url.URL, identityHost string) bool {
	validPath := u.Path == "/oidc/v1/oauth/client/callback" || u.Path == "/oidc/v1/oauth/client/callback/success"
	return u.Scheme == "https" && strings.EqualFold(u.Host, identityHost) && validPath
}

func isUserPage(u *url.URL) bool {
	return u.Scheme == "https" && strings.EqualFold(u.Host, portalHost) &&
		strings.HasPrefix(u.Path, "/content/euda/") && strings.HasSuffix(u.Path, "/user.html")
}

func loginLocation(resp *http.Response) (*url.URL, error) {
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, errors.New(resp.Status)
	}

	switch resp.StatusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		location, err := resp.Location()
		if err != nil {
			return nil, fmt.Errorf("login redirect: %w", err)
		}
		return location, nil
	}

	if resp.Request == nil || resp.Request.URL == nil {
		return nil, errors.New("login response URL missing")
	}
	return resp.Request.URL, nil
}

// get executes a GET request, re-authenticating once on 401/403, and returns the body
func (v *API) get(uri string, headers map[string]string) ([]byte, error) {
	req, err := request.New(http.MethodGet, uri, nil, headers)
	if err != nil {
		return nil, err
	}

	gen := v.loginGen.Load()
	resp, err := v.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()

		if err := v.login(gen); err != nil {
			return nil, fmt.Errorf("login failed: %w", err)
		}

		if req, err = request.New(http.MethodGet, uri, nil, headers); err != nil {
			return nil, err
		}
		if resp, err = v.Do(req); err != nil {
			return nil, err
		}
	}

	return request.ReadBody(resp)
}

// getJSON executes a GET request and decodes the JSON response
func (v *API) getJSON(uri string, res any) error {
	b, err := v.get(uri, map[string]string{"Accept": request.JSONContent})
	if err != nil {
		return err
	}
	return json.Unmarshal(b, res)
}

// Vehicles enumerates the vehicles the user has linked to the portal
func (v *API) Vehicles() ([]Vehicle, error) {
	uri := BaseURL + "/proxy_api/consent/me/vehicles?viewPosition=FRONT_LEFT"

	b, err := v.get(uri, map[string]string{"Accept": request.JSONContent})
	if err != nil {
		return nil, err
	}

	// the response is either a bare array or wrapped in {"vehicles": [...]}
	var arr []Vehicle
	if err := json.Unmarshal(b, &arr); err == nil && len(b) > 0 {
		return arr, nil
	}

	var wrap struct {
		Vehicles []Vehicle `json:"vehicles"`
	}
	if err := json.Unmarshal(b, &wrap); err != nil {
		return nil, err
	}

	return wrap.Vehicles, nil
}

// identifier returns the data-request identifier required for the data delivery calls
func (v *API) identifier(vin string) (string, error) {
	uri := fmt.Sprintf("%s/proxy_api/euda-apim/datarequest/vehicles/%s/metadata/partial", BaseURL, vin)

	var res struct {
		Identifier string `json:"Identifier"`
	}
	if err := v.getJSON(uri, &res); err != nil {
		if se, ok := errors.AsType[*request.StatusError](err); ok && se.HasStatus(404) {
			err = ErrNotConfigured
		}
		return "", err
	}

	if res.Identifier == "" {
		return "", errors.New("no data request configured for vehicle")
	}

	return res.Identifier, nil
}

// datasets lists the available datasets for the given data request
func (v *API) datasets(vin, identifier string) ([]dataset, error) {
	uri := fmt.Sprintf("%s/proxy_api/euda-apim/datadelivery/vehicles/%s/%s/list", BaseURL, vin, identifier)

	b, err := v.get(uri, map[string]string{"Accept": request.JSONContent, "type": "partial"})
	if err != nil {
		// the portal answers 404 "No files available for this request" until the
		// vehicle has delivered its first dataset
		if se, ok := errors.AsType[*request.StatusError](err); ok && se.HasStatus(http.StatusNotFound) {
			return nil, nil
		}
		return nil, err
	}

	var arr []dataset
	if err := json.Unmarshal(b, &arr); err == nil && len(arr) > 0 {
		return arr, nil
	}

	var res struct {
		Files []dataset `json:"files"`
	}
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, err
	}

	return res.Files, nil
}

// download fetches the dataset zip archive
func (v *API) download(vin, identifier, name string) ([]byte, error) {
	uri := fmt.Sprintf("%s/proxy_api/euda-apim/datadelivery/vehicles/%s/%s/download", BaseURL, vin, identifier)
	return v.get(uri, map[string]string{"filename": name, "type": "partial"})
}
