package saic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/evcc-io/evcc/vehicle/saic/requests"
	"golang.org/x/oauth2"
)

const (
	StatRunning = iota
	StatValid
	StatInvalid
)

const (
	RegionEU = "https://gateway-mg-eu.soimt.com/api.app/v1/"
	RegionAU = "https://gateway-mg-au.soimt.com/api.app/v1/"
)

type ConcurrentRequest struct {
	Status int
	Result requests.ChargeStatus
}

// API is an api.Vehicle implementation for SAIC cars
type API struct {
	*request.Helper
	identity *Identity
	request  ConcurrentRequest
	log      *util.Logger
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
	v.request.Status = StatInvalid

	return v
}

func (v *API) doRepeatedRequest(path string, event_id string) error {
	var req *http.Request

	answer := requests.Answer{
		Data: &v.request.Result,
	}

	token, err := v.identity.Token()
	if err != nil {
		v.request.Status = StatInvalid
		return err
	}

	req, err = requests.CreateRequest(
		v.identity.baseUrl,
		path,
		http.MethodGet,
		"",
		request.JSONContent,
		token.AccessToken,
		event_id)
	if err != nil {
		v.request.Status = StatInvalid
		return err
	}

	_, err = v.DoRequest(req, &answer)
	if err == nil {
		v.request.Status = StatValid
	} else if err != api.ErrMustRetry {
		v.request.Status = StatInvalid
	}
	return err
}

// This is running concurrently
func (v *API) repeatRequest(path string, event_id string) {
	var err error
	var count int

	v.request.Status = StatRunning
	for err = api.ErrMustRetry; err == api.ErrMustRetry && count < 20; {
		time.Sleep(2 * time.Second)
		v.log.TRACE.Printf("Starting repeated query. Count: %d", count)
		err = v.doRepeatedRequest(path, event_id)
		count++
	}

	// Make sure that we don't exit here with status running (probably after 20 tries).
	// This would not allow us to ever do a query again.
	if v.request.Status == StatRunning {
		v.request.Status = StatInvalid
	}
	v.log.TRACE.Printf("Exiting repeated query. Count: %d", count)
}

func (v *API) DoRequest(req *http.Request, result *requests.Answer) (string, error) {
	var body []byte

	resp, err := v.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		v.log.TRACE.Printf("DoRequest: %s", resp.Status)
		v.identity.Login()
		return "", api.ErrMustRetry
	}

	event_id := resp.Header.Get("event-id")

	if result != nil {
		body, err = requests.DecryptAnswer(resp)
		if err == nil {
			if err2 := json.Unmarshal(body, result); err2 == nil && result.Code != 0 {
				if result.Code == 4 {
					err = api.ErrMustRetry
				} else {
					err = fmt.Errorf("%d: %s", result.Code, result.Message)
				}
			}
		} else {
			err = fmt.Errorf("decrypt: %w", err)
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
	var req *http.Request
	var err error
	var token *oauth2.Token

	token, err = v.identity.Token()
	if err != nil {
		return err
	}

	path := "vehicle/status?vin=" + requests.Sha256(vin)
	req, err = requests.CreateRequest(
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

	v.DoRequest(req, nil)

	return nil
}

// Status implements the /user/vehicles/<vin>/status api
func (v *API) Status(vin string) (requests.ChargeStatus, error) {
	var req *http.Request
	var res requests.ChargeStatus
	var event_id string
	var err error
	var token *oauth2.Token
	answer := requests.Answer{
		Data: &res,
	}

	// Check if we are already running in the background
	if v.request.Status == StatValid {
		v.request.Status = StatInvalid
		v.log.TRACE.Printf("StatValid. Returning stored value")
		// v.printAnswer()
		return v.request.Result, nil
	}
	if v.request.Status == StatRunning {
		v.log.TRACE.Printf("StatRunning. Exiting")
		return res, api.ErrMustRetry
	}
	v.log.TRACE.Printf("StatInvalid. Starting query")

	token, err = v.identity.Token()
	if err != nil {
		return res, err
	}

	path := "vehicle/charging/mgmtData?vin=" + requests.Sha256(vin)
	// get charging status of vehicle
	req, err = requests.CreateRequest(
		v.identity.baseUrl,
		path,
		http.MethodGet,
		"",
		request.JSONContent,
		token.AccessToken,
		"")
	if err != nil {
		return res, err
	}

	event_id, err = v.DoRequest(req, &answer)
	if err != nil {
		v.log.TRACE.Printf("Getting event id failed")
		return res, err
	}

	if event_id == "" {
		v.log.TRACE.Printf("Answer without event ID")
		return res, api.ErrMustRetry
	}

	req, err = requests.CreateRequest(
		v.identity.baseUrl,
		path,
		http.MethodGet,
		"",
		request.JSONContent,
		token.AccessToken,
		event_id)
	if err != nil {
		v.log.TRACE.Printf("Could not create request %s", err.Error())
		return res, err
	}

	_, err = v.DoRequest(req, &answer)

	// Continue checking....
	if err == api.ErrMustRetry {
		v.request.Status = StatRunning
		v.log.TRACE.Printf(" No answer yet. Continue status query in background")
		go v.repeatRequest(path, event_id)
	} else if err != nil {
		v.log.TRACE.Printf("doRequest failed with %s", err.Error())
	}

	return res, err
}
