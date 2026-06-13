package saic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/evcc-io/evcc/vehicle/saic/requests"
)

const (
	RegionEU = "https://gateway-mg-eu.soimt.com/api.app/v1/"
	RegionAU = "https://gateway-mg-au.soimt.com/api.app/v1/"
)

// API is an api.Vehicle implementation for SAIC cars
type API struct {
	*request.Helper
	identity *Identity
	log      *util.Logger

	mu      sync.Mutex
	running bool                  // background refresh in progress
	valid   bool                  // result holds a valid response
	result  requests.ChargeStatus // last valid response
}

// NewAPI creates a new vehicle
func NewAPI(log *util.Logger, identity *Identity) *API {
	v := &API{
		Helper:   request.NewHelper(log),
		identity: identity,
		log:      log,
	}

	v.Client.Transport = &transport.Decorator{
		Decorator: requests.Decorate,
		Base:      v.Client.Transport,
	}

	return v
}

// store saves a valid response and clears the running flag
func (v *API) store(res requests.ChargeStatus) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.result, v.valid, v.running = res, true, false
}

// last returns the last valid response, or ErrMustRetry until one is available
func (v *API) last() (requests.ChargeStatus, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.valid {
		return v.result, nil
	}
	return v.result, api.ErrMustRetry
}

func (v *API) doRepeatedRequest(path string, event_id string) error {
	token, err := v.identity.Token()
	if err != nil {
		return err
	}

	req, _ := requests.CreateRequest(
		v.identity.baseUrl,
		path,
		http.MethodGet,
		"",
		request.JSONContent,
		token.AccessToken,
		event_id)

	var res requests.Answer[requests.ChargeStatus]
	if _, err = doRequest(v, req, &res); err == nil {
		v.store(res.Data)
	}
	return err
}

// repeatRequest polls for the deferred answer in the background
func (v *API) repeatRequest(path string, event_id string) {
	// always clear running so a future query can start
	defer func() {
		v.mu.Lock()
		v.running = false
		v.mu.Unlock()
	}()

	for count := range 20 {
		time.Sleep(2 * time.Second)
		v.log.TRACE.Printf("repeated query %d", count)
		if err := v.doRepeatedRequest(path, event_id); err != api.ErrMustRetry {
			return
		}
	}
}

func doRequest[T any](v *API, req *http.Request, result *requests.Answer[T]) (string, error) {
	resp, err := v.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		v.log.TRACE.Printf("doRequest: %s", resp.Status)
		v.identity.Login()
		return "", api.ErrMustRetry
	}

	event_id := resp.Header.Get("event-id")

	if result != nil {
		body, err2 := requests.DecodeResponse(resp)
		if err2 != nil {
			return event_id, fmt.Errorf("decrypt: %w", err2)
		}

		v.log.TRACE.Printf("recv: %s", body)

		if err2 := json.Unmarshal(body, result); err2 == nil && result.Code != 0 {
			if result.Code == 4 {
				err = api.ErrMustRetry
			} else {
				err = fmt.Errorf("%d: %s", result.Code, result.Message)
			}
		}
	}

	return event_id, err
}

/* Vehicles implements returns the /user/vehicles api
func (v *API) Vehicles() ([]Vehicle, error) {
	var res []Vehicle
	uri := fmt.Sprintf("%s/eadrax-vcs/v4/vehicles?apptimezone=120&appDateTime=%d", regions[v.region].CocoApiURI, time.Now().UnixMilli())
	err := v.GetJSON(uri, &res)
	return res, err
}
*/

func (v *API) Wakeup(vin string) error {
	token, err := v.identity.Token()
	if err != nil {
		return err
	}

	path := "vehicle/status?vin=" + requests.Sha256(vin)
	req, err := requests.CreateRequest(
		v.identity.baseUrl,
		path,
		http.MethodGet,
		"",
		request.JSONContent,
		token.AccessToken,
		"")
	if err != nil {
		return err
	}

	doRequest[any](v, req, nil)

	return nil
}

// Status implements the /user/vehicles/<vin>/status api
func (v *API) Status(vin string) (requests.ChargeStatus, error) {
	var zero requests.ChargeStatus

	// refresh in progress, return last known value
	v.mu.Lock()
	running := v.running
	v.mu.Unlock()
	if running {
		v.log.TRACE.Printf("query running")
		return v.last()
	}

	token, err := v.identity.Token()
	if err != nil {
		return zero, err
	}

	path := "vehicle/charging/mgmtData?vin=" + requests.Sha256(vin)
	// get charging status of vehicle
	req, _ := requests.CreateRequest(
		v.identity.baseUrl,
		path,
		http.MethodGet,
		"",
		request.JSONContent,
		token.AccessToken,
		"")

	var res requests.Answer[requests.ChargeStatus]
	event_id, err := doRequest(v, req, &res)
	if err != nil {
		return zero, err
	}

	if event_id == "" {
		v.log.TRACE.Printf("answer without event id")
		return v.last()
	}

	req, _ = requests.CreateRequest(
		v.identity.baseUrl,
		path,
		http.MethodGet,
		"",
		request.JSONContent,
		token.AccessToken,
		event_id)

	// answer not yet available, keep polling in the background
	if _, err = doRequest(v, req, &res); err == api.ErrMustRetry {
		v.mu.Lock()
		v.running = true
		v.mu.Unlock()
		v.log.TRACE.Printf("no answer yet, continuing in background")
		go v.repeatRequest(path, event_id)
		return v.last()
	} else if err != nil {
		return zero, err
	}

	v.store(res.Data)
	return res.Data, nil
}
