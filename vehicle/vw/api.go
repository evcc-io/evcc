package vw

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// DefaultBaseURI is the VW api base URI
const DefaultBaseURI = "https://msg.volkswagen.de/fs-car"

// RegionAPI is the VW api used for determining the home region
const RegionAPI = "https://mal-1a.prd.ece.vwg-connect.com/api"

// API is the VW api client
type API struct {
	*request.Helper
	brand, country string
	baseURI        string
	statusURI      string
	log            *util.Logger
	ts             oauth2.TokenSource
}

// NewAPI creates a new api client
func NewAPI(log *util.Logger, ts oauth2.TokenSource, brand, country string) *API {
	v := &API{
		Helper:  request.NewHelper(log),
		brand:   brand,
		country: country,
		baseURI: DefaultBaseURI,
		log:     log,
		ts:      ts,
	}

	v.Client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   v.Client.Transport,
	}

	return v
}

// ensureValidToken checks if token is valid and refreshes if needed
func (v *API) ensureValidToken(ctx context.Context) error {
	token, err := v.ts.Token()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// Check if token is expired or will expire within 5 minutes
	if token.Expiry.IsZero() || time.Now().Add(5*time.Minute).After(token.Expiry) {
		v.log.DEBUG.Println("VW token expired or expiring soon, refreshing proactively")
		// Force token refresh by requesting new token
		if _, err := v.ts.Token(); err != nil {
			return fmt.Errorf("proactive token refresh failed: %w", err)
		}
		v.log.DEBUG.Println("VW token refreshed successfully")
	}

	return nil
}

// doWithRetry executes a function and retries once on HTTP 400 error after forcing token refresh
func (v *API) doWithRetry(ctx context.Context, fn func() error) error {
	// Proactively ensure token is valid before making API call
	if err := v.ensureValidToken(ctx); err != nil {
		v.log.WARN.Printf("VW token validation failed: %v", err)
	}

	err := fn()
	if err == nil {
		return nil
	}

	// Check if it's a 400 error
	var se *request.StatusError
	if errors.As(err, &se) && se.StatusCode() == http.StatusBadRequest {
		v.log.DEBUG.Printf("VW API returned 400 (Bad Request), attempting token refresh")

		// Force a new token request - the TokenSource will handle refresh
		if _, tokenErr := v.ts.Token(); tokenErr != nil {
			v.log.WARN.Printf("VW token refresh failed: %v", tokenErr)
			return fmt.Errorf("token refresh after 400 error failed: %w", tokenErr)
		}

		v.log.DEBUG.Println("VW token refreshed, retrying API call")

		// Retry the operation once
		if retryErr := fn(); retryErr == nil {
			return nil
		} else {
			v.log.DEBUG.Printf("VW API retry after token refresh also failed: %v", retryErr)
			return retryErr
		}
	}

	return err
}

// logAPIError logs detailed error information for debugging
func (v *API) logAPIError(err error, operation string) {
	if err == nil {
		return
	}

	var se *request.StatusError
	if errors.As(err, &se) {
		resp := se.Response()
		v.log.DEBUG.Printf("VW API %s failed: status=%d (%s), url=%s",
			operation,
			se.StatusCode(),
			http.StatusText(se.StatusCode()),
			resp.Request.URL.String(),
		)

		// Log specific guidance for common errors
		switch se.StatusCode() {
		case http.StatusBadRequest:
			v.log.WARN.Printf("VW API 400 error in %s - possible token expiration or invalid request", operation)
		case http.StatusUnauthorized:
			v.log.WARN.Printf("VW API 401 error in %s - authentication failed", operation)
		case http.StatusTooManyRequests:
			v.log.WARN.Printf("VW API 429 error in %s - rate limit exceeded", operation)
		}
	} else {
		v.log.DEBUG.Printf("VW API %s failed: %v", operation, err)
	}
}

// HomeRegion updates the home region for the given vehicle
func (v *API) HomeRegion(vin string) error {
	return v.doWithRetry(context.Background(), func() error {
		var res HomeRegion
		uri := fmt.Sprintf("%s/cs/vds/v1/vehicles/%s/homeRegion", RegionAPI, vin)

		err := v.GetJSON(uri, &res)
		if err == nil {
			if api := res.HomeRegion.BaseURI.Content; strings.HasPrefix(api, "https://mal-3a.prd.eu.dp.vwg-connect.com") {
				api = "https://fal" + strings.TrimPrefix(api, "https://mal")
				api = strings.TrimSuffix(api, "/api") + "/fs-car"
				v.baseURI = api
			}
		} else {
			v.logAPIError(err, "HomeRegion")
			if res.Error != nil {
				err = res.Error.Error()
			}
		}

		return err
	})
}

// RolesRights implements the /rolesrights/operationlist response
func (v *API) RolesRights(vin string) (res RolesRights, err error) {
	err = v.doWithRetry(context.Background(), func() error {
		uri := fmt.Sprintf("%s/rolesrights/operationlist/v3/vehicles/%s", RegionAPI, vin)
		if apiErr := v.GetJSON(uri, &res); apiErr != nil {
			v.logAPIError(apiErr, "RolesRights")
			return apiErr
		}
		return nil
	})
	return res, err
}

// ServiceURI renders the service URI for the given vin and service
func (v *API) ServiceURI(vin, service string, rr RolesRights) (uri string) {
	if si := rr.ServiceByID(service); si != nil {
		uri = si.InvocationUrl.Content
		uri = strings.ReplaceAll(uri, "{vin}", vin)
		uri = strings.ReplaceAll(uri, "{brand}", v.brand)
		uri = strings.ReplaceAll(uri, "{country}", v.country)
	}

	return uri
}

// Status implements the /status response
func (v *API) Status(vin string) (res StatusResponse, err error) {
	err = v.doWithRetry(context.Background(), func() error {
		uri := fmt.Sprintf("%s/bs/vsr/v1/vehicles/%s/status", RegionAPI, vin)
		if v.statusURI != "" {
			uri = v.statusURI
		}

		headers := map[string]string{
			"Accept":        request.JSONContent,
			"X-App-Name":    "foo", // required
			"X-App-Version": "foo", // required
		}

		req, reqErr := request.New(http.MethodGet, uri, nil, headers)
		if reqErr != nil {
			return reqErr
		}

		apiErr := v.DoJSON(req, &res)

		if se := new(request.StatusError); errors.As(apiErr, &se) {
			var rr RolesRights
			rr, apiErr = v.RolesRights(vin)

			if apiErr == nil {
				if uri = v.ServiceURI(vin, StatusService, rr); uri == "" {
					apiErr = fmt.Errorf("%s not found", StatusService)
				}
			}

			if apiErr == nil {
				if strings.HasSuffix(uri, fmt.Sprintf("%s/", vin)) {
					uri += "status"
				}

				if req, apiErr = request.New(http.MethodGet, uri, nil, headers); apiErr == nil {
					if apiErr = v.DoJSON(req, &res); apiErr == nil {
						v.statusURI = uri
					}
				}
			}
		}

		if apiErr != nil {
			v.logAPIError(apiErr, "Status")
		}

		return apiErr
	})

	return res, err
}

// Charger implements the /charger response
func (v *API) Charger(vin string) (res ChargerResponse, err error) {
	err = v.doWithRetry(context.Background(), func() error {
		uri := fmt.Sprintf("%s/bs/batterycharge/v1/%s/%s/vehicles/%s/charger", v.baseURI, v.brand, v.country, vin)
		if apiErr := v.GetJSON(uri, &res); apiErr != nil {
			v.logAPIError(apiErr, "Charger")
			if res.Error != nil {
				return res.Error.Error()
			}
			return apiErr
		}
		return nil
	})
	return res, err
}

// Climater implements the /climater response
func (v *API) Climater(vin string) (res ClimaterResponse, err error) {
	err = v.doWithRetry(context.Background(), func() error {
		uri := fmt.Sprintf("%s/bs/climatisation/v1/%s/%s/vehicles/%s/climater", v.baseURI, v.brand, v.country, vin)
		if apiErr := v.GetJSON(uri, &res); apiErr != nil {
			v.logAPIError(apiErr, "Climater")
			if res.Error != nil {
				return res.Error.Error()
			}
			return apiErr
		}
		return nil
	})
	return res, err
}

// Position implements the /position response
func (v *API) Position(vin string) (res PositionResponse, err error) {
	err = v.doWithRetry(context.Background(), func() error {
		uri := fmt.Sprintf("%s/bs/cf/v1/%s/%s/vehicles/%s/position", v.baseURI, v.brand, v.country, vin)

		req, reqErr := request.New(http.MethodGet, uri, nil, map[string]string{
			"Accept":        request.JSONContent,
			"Content-type":  "application/vnd.vwg.mbb.carfinderservice_v1_0_0+json",
			"X-App-Name":    "foo", // required
			"X-App-Version": "foo", // required
		})

		if reqErr != nil {
			return reqErr
		}

		if apiErr := v.DoJSON(req, &res); apiErr != nil {
			v.logAPIError(apiErr, "Position")
			if res.Error != nil {
				return res.Error.Error()
			}
			return apiErr
		}
		return nil
	})
	return res, err
}

const (
	ActionCharge      = "batterycharge"
	ActionChargeStart = "start"
	ActionChargeStop  = "stop"
)

type actionDefinition struct {
	contentType string
	appendix    string
}

var actionDefinitions = map[string]actionDefinition{
	ActionCharge: {
		"application/vnd.vwg.mbb.ChargerAction_v1_0_0+xml",
		"charger/actions",
	},
}

// Action implements vehicle actions
func (v *API) Action(vin, action, value string) error {
	return v.doWithRetry(context.Background(), func() error {
		def := actionDefinitions[action]

		uri := fmt.Sprintf("%s/bs/%s/v1/%s/%s/vehicles/%s/%s", v.baseURI, action, v.brand, v.country, vin, def.appendix)
		body := "<?xml version=\"1.0\" encoding=\"UTF-8\" ?><action><type>" + value + "</type></action>"

		req, reqErr := request.New(http.MethodPost, uri, strings.NewReader(body), map[string]string{
			"Content-type": def.contentType,
		})

		if reqErr != nil {
			return reqErr
		}

		var resp *http.Response
		var err error
		if resp, err = v.Do(req); err == nil {
			resp.Body.Close()
		} else {
			v.logAPIError(err, fmt.Sprintf("Action(%s,%s)", action, value))
		}

		return err
	})
}

// Any implements any api response
func (v *API) Any(base, vin string) (any, error) {
	var res any
	uri := fmt.Sprintf("%s/"+strings.TrimLeft(base, "/"), v.baseURI, v.brand, v.country, vin)
	err := v.GetJSON(uri, &res)
	return res, err
}
