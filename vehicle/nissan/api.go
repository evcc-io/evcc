package nissan

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

const (
	refreshTimeout = 5 * time.Minute
	statusExpiry   = time.Minute
)

type API struct {
	*request.Helper
	VIN         string
	refreshID   string
	refreshTime time.Time
}

func NewAPI(log *util.Logger, identity oauth2.TokenSource, vin string) *API {
	v := &API{
		Helper: request.NewHelper(log),
		VIN:    vin,
	}

	// api is unbelievably slow when retrieving status
	v.Client.Timeout = 120 * time.Second

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
		uri := fmt.Sprintf("%s/v4/users/%s/cars", UserBaseURL, user.UserID)
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

const timeFormat = "2006-01-02T15:04:05Z"

// Battery provides battery api response
func (v *API) Battery() (Response, error) {
	// request battery status
	uri := fmt.Sprintf("%s/v1/cars/%s/battery-status", CarAdapterBaseURL, v.VIN)

	var res Response
	err := v.GetJSON(uri, &res)

	var ts time.Time
	if err == nil {
		ts, err = time.Parse(timeFormat, res.Data.Attributes.LastUpdateTime)

		// return the current value
		if time.Since(ts) <= statusExpiry {
			v.refreshID = ""
			return res, err
		}
	}

	// request a refresh, irrespective of a previous error
	if v.refreshID == "" {
		if err = v.refreshRequest(); err == nil {
			err = api.ErrMustRetry
		}

		return res, err
	}

	// refresh finally expired
	if time.Since(v.refreshTime) > refreshTimeout {
		v.refreshID = ""
		if err == nil {
			err = api.ErrTimeout
		}
	} else {
		if len(res.Errors) > 0 {
			// extract error code
			e := res.Errors[0]
			err = fmt.Errorf("%s: %s", e.Code, e.Detail)
		} else {
			// wait for refresh, irrespective of a previous error
			err = api.ErrMustRetry
		}
	}

	return res, err
}

// refreshRequest requests  battery status refresh
func (v *API) refreshRequest() error {
	uri := fmt.Sprintf("%s/v1/cars/%s/actions/refresh-battery-status", CarAdapterBaseURL, v.VIN)

	data := Request{
		Data: Payload{
			Type: "RefreshBatteryStatus",
		},
	}

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), map[string]string{
		"Content-Type": "application/vnd.api+json",
	})

	var res Response
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	if err == nil {
		v.refreshID = res.Data.ID
		v.refreshTime = time.Now()

		if v.refreshID == "" {
			err = errors.New("refresh failed")
		}
	}

	return err
}

type Action string

const (
	ActionChargeStart Action = "start"
	ActionChargeStop  Action = "stop"
)

// ChargingAction provides actions/charging-start api response
func (v *API) ChargingAction(action Action) (Response, error) {
	uri := fmt.Sprintf("%s/v1/cars/%s/actions/charging-start", CarAdapterBaseURL, v.VIN)

	data := Request{
		Data: Payload{
			Type: "ChargingStart",
			Attributes: map[string]interface{}{
				"action": action,
			},
		},
	}

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), map[string]string{
		"Content-Type": "application/vnd.api+json",
	})

	var res Response
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}
