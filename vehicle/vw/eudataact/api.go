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

	if err := v.login(); err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	apiReg[key] = v

	return v, nil
}

// login performs the OIDC authorization-code flow against the VW identity
// service. The portal relies on the session cookies that are set while the
// browser follows the redirect chain back to RedirectURI, so the cookie jar is
// kept on the client for all subsequent data calls.
func (v *API) login() error {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return err
	}

	v.Client.Jar = jar
	v.Client.CheckRedirect = func(req *http.Request, _ []*http.Request) error {
		// follow the https redirect chain to the portal, stop at app schemes
		if req.URL.Scheme != "https" {
			return http.ErrUseLastResponse
		}
		return nil
	}

	// prime the portal session (best effort)
	if resp, err := v.Get(BaseURL + "/"); err == nil {
		resp.Body.Close()
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

	resp, err := v.Get(vwidentity.Config.AuthURL + "?" + q.Encode())
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
	resp, err = v.PostForm(uri, url.Values{
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
	resp, err = v.PostForm(uri, url.Values{
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

	if resp.StatusCode >= http.StatusBadRequest {
		return errors.New(resp.Status)
	}

	// a successful login lands on the portal; a remaining signin/consent url means
	// the user has not completed the one-time browser consent and vehicle linking
	final := resp.Request.URL
	if strings.Contains(final.Path, "signin-service") || strings.Contains(final.Path, "/consent") || strings.Contains(final.Path, "/error") {
		return api.UrlError(
			fmt.Sprintf("login did not complete- open the portal and confirm consent: %s", final),
			final,
		)
	}
	if final.Host != portalHost {
		return fmt.Errorf("login did not complete: unexpected landing host %s", final.Host)
	}

	return nil
}

// get executes a GET request, re-authenticating once on 401/403, and returns the body
func (v *API) get(uri string, headers map[string]string) ([]byte, error) {
	req, err := request.New(http.MethodGet, uri, nil, headers)
	if err != nil {
		return nil, err
	}

	resp, err := v.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()

		if err := v.login(); err != nil {
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
	if err := json.Unmarshal(b, &arr); err == nil && len(arr) > 0 {
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
