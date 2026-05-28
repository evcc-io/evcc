package weconnect

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
	"github.com/google/uuid"
	"golang.org/x/net/publicsuffix"
)

// VW's cariad BFF sits behind an Azure WAF that 403s anything sending evcc's
// default User-Agent or extra OAuth parameters. The official Volkswagen
// Android app and the volkswagencarnet Python client both reach the same
// endpoint with browser-shaped headers and just redirect_uri + nonce in the
// query; the server then issues its own Auth0 state via the redirect chain.
//
// See https://github.com/robinostlund/volkswagencarnet/pull/329 for the flow
// this implementation mirrors.
var plainHeaders = map[string]string{
	"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
	"Accept-Encoding": "gzip, deflate",
	"Connection":      "keep-alive",
	"User-Agent":      "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
}

// Login performs the WeConnect BFF login and returns the url.Values expected
// by loginapps.Service.Exchange (state, id_token, access_token, code).
func Login(log *util.Logger, user, password string) (url.Values, error) {
	helper := request.NewHelper(log)

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}
	helper.Client.Jar = jar
	defer func() { helper.Client.Jar = nil }()

	// Don't follow redirects automatically. We need to inspect each Location
	// header to extract Auth0's state and to stop at the weconnect:// callback.
	helper.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	// Step 1: authorize preamble. The BFF replies with a 302 to the Auth0
	// login URL, embedding its own state in the Location query.
	q := url.Values{
		"redirect_uri": {"weconnect://authenticated"},
		"nonce":        {uuid.NewString()},
	}
	preambleURI := LoginURL + "?" + q.Encode()

	loc, err := getRedirect(helper, preambleURI)
	if err != nil {
		return nil, fmt.Errorf("preamble: %w", err)
	}

	parsedLoc, err := url.Parse(loc)
	if err != nil {
		return nil, fmt.Errorf("preamble location: %w", err)
	}
	auth0State := parsedLoc.Query().Get("state")
	if auth0State == "" {
		return nil, errors.New("preamble: no state in redirect")
	}

	// Step 2: follow the redirect chain until we land on the login page (200).
	current, err := url.Parse(preambleURI)
	if err != nil {
		return nil, err
	}
	current = current.ResolveReference(parsedLoc)

	var body []byte
	for range 10 {
		req, err := request.New(http.MethodGet, current.String(), nil, plainHeaders)
		if err != nil {
			return nil, err
		}
		resp, err := helper.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusOK {
			body, err = io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return nil, err
			}
			break
		}

		resp.Body.Close()
		if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
			return nil, fmt.Errorf("login page redirect: status %d", resp.StatusCode)
		}

		next, err := url.Parse(resp.Header.Get("Location"))
		if err != nil {
			return nil, err
		}
		current = current.ResolveReference(next)
	}
	if body == nil {
		return nil, errors.New("login page: too many redirects")
	}

	// Step 3: extract the Auth0 form state token.
	formState, err := vwidentity.ExtractState(body)
	if err != nil {
		return nil, err
	}

	// Step 4: POST credentials. Auth0 rejects the request without a Referer
	// matching the login URL and an Origin pointing at identity.vwgroup.io.
	postURI := vwidentity.BaseURL + "/u/login?state=" + formState
	postHeaders := map[string]string{
		"Accept":          plainHeaders["Accept"],
		"Accept-Encoding": plainHeaders["Accept-Encoding"],
		"Connection":      plainHeaders["Connection"],
		"User-Agent":      plainHeaders["User-Agent"],
		"Content-Type":    request.FormContent,
		"Referer":         current.String(),
		"Origin":          vwidentity.BaseURL,
	}
	loginData := url.Values{
		"username": {user},
		"password": {password},
		"state":    {formState},
	}

	req, err := request.New(http.MethodPost, postURI, strings.NewReader(loginData.Encode()), postHeaders)
	if err != nil {
		return nil, err
	}
	resp, err := helper.Do(req)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("login: %s", resp.Status)
	}

	// Step 5: follow redirects until we hit the weconnect:// callback URL.
	final, err := followCallback(helper, current, resp.Header.Get("Location"))
	if err != nil {
		return nil, err
	}

	q, err = vwidentity.ParseAuthLocation(final)
	if err != nil {
		return nil, err
	}

	// Ensure state is present (the server should echo Auth0's state into the
	// callback fragment, but fall back to the one captured in step 1).
	if q.Get("state") == "" && auth0State != "" {
		q.Set("state", auth0State)
	}

	return q, nil
}

// getRedirect issues a GET with plainHeaders and returns the Location header
// of a 302/303 response. Any other status is an error.
func getRedirect(helper *request.Helper, uri string) (string, error) {
	req, err := request.New(http.MethodGet, uri, nil, plainHeaders)
	if err != nil {
		return "", err
	}
	resp, err := helper.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", errors.New("missing Location header")
	}
	return loc, nil
}

// followCallback walks the redirect chain after credential POST until the
// scheme switches away from https (i.e. weconnect://authenticated).
func followCallback(helper *request.Helper, base *url.URL, location string) (*url.URL, error) {
	current := base
	loc := location

	for range 10 {
		next, err := url.Parse(loc)
		if err != nil {
			return nil, err
		}
		current = current.ResolveReference(next)

		if current.Scheme != "https" {
			return current, nil
		}

		req, err := request.New(http.MethodGet, current.String(), nil, plainHeaders)
		if err != nil {
			return nil, err
		}
		resp, err := helper.Do(req)
		if err != nil {
			return nil, err
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return nil, errors.New("callback: stopped at https 200, expected weconnect:// redirect")
		}
		if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
			return nil, fmt.Errorf("callback: unexpected status %d", resp.StatusCode)
		}
		loc = resp.Header.Get("Location")
		if loc == "" {
			return nil, errors.New("callback: missing Location header")
		}
	}
	return nil, errors.New("callback: too many redirects")
}
