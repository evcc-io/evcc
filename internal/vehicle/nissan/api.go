package nissan

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/andig/evcc/internal/vehicle/kamereon"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"golang.org/x/oauth2"
)

type API struct {
	*request.Helper
	VIN string
}

func NewAPI(log *util.Logger, identity oauth2.TokenSource, vin string) *API {
	v := &API{
		Helper: request.NewHelper(log),
		VIN:    vin,
	}

	// replace client transport with authenticated transport
	v.Client.Transport = &oauth2.Transport{
		Source: identity,
		Base:   v.Client.Transport,
	}

	return v
}

func (v *API) Vehicles() ([]string, error) {
	var user struct{ UserID string }
	uri := fmt.Sprintf("%s/v1/users/current", UserAdapterBaseURL)
	err := v.GetJSON(uri, &user)

	var res Vehicles
	if err == nil {
		uri := fmt.Sprintf("%s/v2/users/%s/cars", UserBaseURL, user.UserID)
		err = v.GetJSON(uri, &res)
	}

	var vehicles []string
	if err == nil {
		for _, v := range res.Data {
			vehicles = append(vehicles, v.VIN)
		}
	}

	return vehicles, err
}

// Battery provides battery api response
func (v *API) Battery() (interface{}, error) {
	// refresh battery status
	uri := fmt.Sprintf("%s/v1/cars/%s/actions/refresh-battery-status", CarAdapterBaseURL, v.VIN)

	data := strings.NewReader(`{"data": {"type": "RefreshBatteryStatus"}}`)
	req, err := request.New(http.MethodPost, uri, data, map[string]string{
		"Content-Type": "application/vnd.api+json",
	})

	var res kamereon.Response
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	// request battery status
	if err == nil {
		uri = fmt.Sprintf("%s/v1/cars/%s/battery-status", CarAdapterBaseURL, v.VIN)
		err = v.GetJSON(uri, &res)
	}

	return res, err
}
