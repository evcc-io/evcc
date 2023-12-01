package bmw

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

// https://github.com/bimmerconnected/bimmer_connected
// https://github.com/TA2k/ioBroker.bmw

// API is an api.Vehicle implementation for BMW cars
type API struct {
	*request.Helper
	region string
}

// NewAPI creates a new vehicle
func NewAPI(log *util.Logger, brand, region string, identity oauth2.TokenSource) *API {
	v := &API{
		Helper: request.NewHelper(log),
		region: strings.ToUpper(region),
	}

	// replace client transport with authenticated transport
	v.Client.Transport = &transport.Decorator{
		Base: &oauth2.Transport{
			Source: identity,
			Base:   v.Client.Transport,
		},
		Decorator: transport.DecorateHeaders(map[string]string{
			"X-User-Agent": fmt.Sprintf("android(SP1A.210812.016.C1);%s;99.0.0(99999);row", brand),
		}),
	}

	return v
}

// Vehicles implements returns the /user/vehicles api
func (v *API) Vehicles() ([]Vehicle, error) {
	var res []Vehicle
	uri := fmt.Sprintf("%s/eadrax-vcs/v4/vehicles?apptimezone=120&appDateTime=%d", regions[v.region].CocoApiURI, time.Now().UnixMilli())
	err := v.GetJSON(uri, &res)
	return res, err
}

// Status implements the /user/vehicles/<vin>/status api
func (v *API) Status(vin string) (VehicleStatus, error) {
	var res VehicleStatus
	uri := fmt.Sprintf("%s/eadrax-vcs/v4/vehicles/state?apptimezone=120&appDateTime=%d", regions[v.region].CocoApiURI, time.Now().UnixMilli())

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"bmw-vin": vin,
	})
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}

const (
	CHARGE_START = "start-charging"
	CHARGE_STOP  = "stop-charging"
	DOOR_LOCK    = "door-lock"
	LIGHT_FLASH  = "light-flash"

	REMOTE_SERVICE_BASE_URL   = "eadrax-vrccs/v3/presentation/remote-commands"
	VEHICLE_CHARGING_BASE_URL = "eadrax-crccs/v1/vehicles"
)

var serviceUrls = map[string]string{
	CHARGE_START: VEHICLE_CHARGING_BASE_URL,
	CHARGE_STOP:  VEHICLE_CHARGING_BASE_URL,
}

type Event struct {
	EventID      string
	CreationTime time.Time
}

// Action implements the /remote-commands/<vin>/<service> api
func (v *API) Action(vin, action string) (Event, error) {
	var res Event

	path, ok := serviceUrls[action]
	if !ok {
		path = REMOTE_SERVICE_BASE_URL
	}
	uri := fmt.Sprintf("%s/%s/%s/%s", regions[v.region].CocoApiURI, path, vin, action)

	req, err := request.New(http.MethodPost, uri, nil, map[string]string{
		"Accept": request.JSONContent,
	})

	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}
