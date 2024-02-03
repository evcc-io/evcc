package saic

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/evcc-io/evcc/vehicle/saic/requests"
	"golang.org/x/oauth2"
)

// API is an api.Vehicle implementation for SAIC cars
type API struct {
	*request.Helper
	identity *Identity
	Logger   *util.Logger
}

// NewAPI creates a new vehicle
func NewAPI(log *util.Logger, identity *Identity) *API {
	v := &API{
		Helper:   request.NewHelper(log),
		identity: identity,
		Logger:   log,
	}

	v.Client.Transport = &transport.Decorator{
		Decorator: requests.Decorate,
		Base:      v.Client.Transport,
	}

	return v
}

func (v *API) DoRequest(req *http.Request, result *requests.Answer) (string, error) {
	var body []byte

	resp, err := v.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		v.Logger.DEBUG.Printf("DoRequest: %s", resp.Status)
		v.identity.Login()
		return "", api.ErrMustRetry
	}

	event_id := resp.Header.Get("event-id")

	if result != nil {
		body, err = requests.DecryptAnswer(resp)
		if err == nil {
			err = json.Unmarshal(body, result)
			if err == nil && result.Code != 0 {
				if result.Code == 4 {
					err = api.ErrMustRetry
				} else {
					err = fmt.Errorf("%d: %s\n", result.Code, result.Message)
				}
				v.Logger.DEBUG.Printf("%d: %s\n", result.Code, result.Message)
			}
		} else {
			if err != nil {
				v.Logger.DEBUG.Printf("Decrypt: %s", err.Error())
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
	var req *http.Request
	var err error
	var token *oauth2.Token

	token, err = v.identity.Token()
	if err != nil {
		return err
	}

	url := requests.BASE_URL_P + "vehicle/status?vin=" + requests.Sha256(vin)
	req, err = requests.CreateRequest(url,
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

	token, err = v.identity.Token()
	if err != nil {
		return res, err
	}

	url := requests.BASE_URL_P + "vehicle/charging/mgmtData?vin=" + requests.Sha256(vin)

	// get charging status of vehicle
	req, err = requests.CreateRequest(url,
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
		v.Logger.DEBUG.Printf("Getting event id failed")
		return res, err
	}

	if event_id == "" {
		v.Logger.ERROR.Printf("Answer without event ID")
		return res, api.ErrMustRetry
	}

	req, err = requests.CreateRequest(url,
		http.MethodGet,
		"",
		request.JSONContent,
		token.AccessToken,
		event_id)
	if err != nil {
		return res, err
	}

	_, err = v.DoRequest(req, &answer)

	return res, err
}
