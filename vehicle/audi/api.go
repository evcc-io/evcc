package audi

import (
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/vw"
	"golang.org/x/oauth2"
)

// DefaultBaseURI is the Audi api base URI
const DefaultBaseURI = "https://msg.audi.de/fs-car"

// API is the VW api client
type API struct {
	*request.Helper
	brand, country string
	baseURI        string
}

// NewAPI creates a new api client
func NewAPI(log *util.Logger, ts oauth2.TokenSource, brand, country string) *API {
	v := &API{
		Helper:  request.NewHelper(log),
		brand:   brand,
		country: country,
		baseURI: DefaultBaseURI,
	}

	v.Client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   v.Client.Transport,
	}

	return v
}

func (v *API) getJSON(uri string, res interface{}) error {
	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept":        request.JSONContent,
		"X-App-Name":    "myAudi",
		"X-Country-Id":  "DE",
		"X-Language-Id": "de",
		"X-App-Version": "3.22.0",
	})

	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return err
}

// Status returns the /vehicles/<vin>/status response
func (v *API) Status(vin string) (vw.StatusResponse, error) {
	var res vw.StatusResponse

	uri := fmt.Sprintf("%s/bs/vsr/v1/Audi/DE/vehicles/%s/status", DefaultBaseURI, vin)
	// uri := fmt.Sprintf("https://mal-3a.prd.eu.dp.vwg-connect.com/api/bs/vsr/v1/vehicles/%s/status", vin)
	err := v.getJSON(uri, &res)

	if err != nil && res.Error.ErrorCode != "" {
		err = fmt.Errorf("%w (%s: %s)", err, res.Error.ErrorCode, res.Error.Description)
	}

	return res, err
}
