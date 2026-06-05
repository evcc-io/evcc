package eudab2t

import (
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// BaseURL is the EU Data Act B2T (business-to-third-party) pull api endpoint.
const BaseURL = "https://api.drivesomethinggreater.com/euda-b2t-pull"

// API is a client for the EU Data Act B2T pull api.
//
// Unlike the consumer portal client in vehicle/vw/eudataact this is a plain REST
// service: the data consumer authenticates with a marketplace api key issued to
// it and pulls the latest data points per vin. There is no browser consent flow
// and no zipped dataset download.
type API struct {
	*request.Helper
	baseURL string
	apiKey  string
}

// NewAPI returns a B2T pull client authenticated with the given marketplace api key.
func NewAPI(log *util.Logger, apiKey string) *API {
	return &API{
		Helper:  request.NewHelper(log),
		baseURL: BaseURL,
		apiKey:  apiKey,
	}
}

// headers returns the marketplace api key header sent with every request
func (v *API) headers() map[string]string {
	return map[string]string{
		"x-marketplace-api-key": v.apiKey,
		"Accept":                request.JSONContent,
	}
}

// Data returns the latest data points for the vehicle as a flat map of EU Data
// Act field names to their string values.
func (v *API) Data(vin string) (map[string]string, error) {
	uri := fmt.Sprintf("%s/subscription/vehicles/%s/data", v.baseURL, vin)

	req, err := request.New(http.MethodGet, uri, nil, v.headers())
	if err != nil {
		return nil, err
	}

	var res struct {
		Data map[string]string `json:"data"`
	}
	if err := v.DoJSON(req, &res); err != nil {
		return nil, err
	}

	return res.Data, nil
}

// Subscribe adds and/or removes vins from the marketplace subscription.
func (v *API) Subscribe(add, remove []string) error {
	uri := v.baseURL + "/subscription/vehicles"

	body := request.MarshalJSON(VINList{Add: add, Remove: remove})

	req, err := request.New(http.MethodPost, uri, body, v.headers(), request.JSONEncoding)
	if err != nil {
		return err
	}

	// the endpoint acknowledges with an empty 200 body
	_, err = v.DoBody(req)
	return err
}

// Status lists the subscription's vehicles and their consent status. In addition
// to the marketplace api key it requires the user's identity (IDK) token.
func (v *API) Status(token string) ([]ConsentInfo, error) {
	uri := v.baseURL + "/subscription/vehicles/status"

	headers := v.headers()
	headers["Authorization"] = token

	req, err := request.New(http.MethodGet, uri, nil, headers)
	if err != nil {
		return nil, err
	}

	var res []ConsentInfo
	if err := v.DoJSON(req, &res); err != nil {
		return nil, err
	}

	return res, nil
}
