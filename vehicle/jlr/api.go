package jlr

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

const (
	IF9_BASE_URL  = "https://if9.prod-row.jlrmotor.com/if9/jlr"
	IFOP_BASE_URL = "https://ifop.prod-row.jlrmotor.com/ifop/jlr"
)

// API is the Jaguar/Landrover api client
type API struct {
	*request.Helper
}

// NewAPI creates a new api client
func NewAPI(log *util.Logger, device string, ts oauth2.TokenSource) *API {
	v := &API{
		Helper: request.NewHelper(log),
	}

	// v.Client.Transport = &transport.Decorator{
	// 	Base: &oauth2.Transport{
	// 		Source: oauth2.StaticTokenSource(&t.Token),
	// 		Base:   v.Transport,
	// 	},
	// 	Decorator: transport.DecorateHeaders(map[string]string{
	// 		"X-Device-Id":             device,
	// 		"x-telematicsprogramtype": "jlrpy",
	// 	}),
	// }

	v.Client.Transport = &transport.Decorator{
		Decorator: func(req *http.Request) error {
			token, err := ts.Token()
			if err == nil {
				for k, v := range map[string]string{
					"Authorization":           fmt.Sprintf("Bearer %s", token.AccessToken),
					"Content-type":            request.JSONContent,
					"X-Device-Id":             device,
					"x-telematicsprogramtype": "jlrpy",
				} {
					req.Header.Set(k, v)
				}
			}
			return err
		},
		Base: v.Client.Transport,
	}

	return v
}

func (v *API) User(name string) (User, error) {
	var res User

	uri := fmt.Sprintf("%s/users?loginName=%s", IF9_BASE_URL, url.QueryEscape(name))
	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/vnd.wirelesscar.ngtp.if9.User-v3+json",
	})

	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}

func (v *API) Vehicles(user string) ([]string, error) {
	var vehicles []string
	var resp VehiclesResponse

	uri := fmt.Sprintf("%s/users/%s/vehicles?primaryOnly=true", IF9_BASE_URL, user)

	err := v.GetJSON(uri, &resp)
	if err == nil {
		for _, v := range resp.Vehicles {
			vehicles = append(vehicles, v.VIN)
		}
	}

	return vehicles, nil
}

// Status returns the vehicle status
func (v *API) Status(vin string) (StatusResponse, error) {
	var status StatusResponse

	uri := fmt.Sprintf("%s/vehicles/%s/status?includeInactive=true", IF9_BASE_URL, vin)
	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/vnd.ngtp.org.if9.healthstatus-v3+json",
	})

	if err == nil {
		err = v.DoJSON(req, &status)
	}

	return status, err
}
