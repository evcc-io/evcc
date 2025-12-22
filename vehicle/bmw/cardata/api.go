package cardata

import (
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

const ApiURL = "https://api-cardata.bmwgroup.com"

// requiredKeys are the necessary data dictionary entities according to
// https://mybmwweb-utilities.api.bmw/de-de/utilities/bmw/api/cd/catalogue/file
var requiredKeys = []string{
	"vehicle.body.chargingPort.status",
	"vehicle.cabin.hvac.preconditioning.status.comfortState",
	"vehicle.drivetrain.batteryManagement.header",
	"vehicle.drivetrain.electricEngine.charging.hvStatus",
	"vehicle.drivetrain.electricEngine.charging.status",
	"vehicle.drivetrain.electricEngine.charging.timeRemaining",
	"vehicle.drivetrain.electricEngine.kombiRemainingElectricRange",
	"vehicle.powertrain.electric.battery.stateOfCharge.target",
	"vehicle.vehicle.preConditioning.activity",
	"vehicle.vehicle.travelledDistance",
}

const requiredVersion = "v3"

type API struct {
	*request.Helper
}

func NewAPI(log *util.Logger, ts oauth2.TokenSource) *API {
	v := &API{
		Helper: request.NewHelper(log),
	}

	v.Transport = &oauth2.Transport{
		Source: ts,
		Base: &transport.Decorator{
			Decorator: transport.DecorateHeaders(map[string]string{
				"x-version": "v1",
			}),
			Base: v.Transport,
		},
	}

	return v
}

func (v *API) Vehicles() ([]string, error) {
	var res []VehicleMapping
	err := v.GetJSON(ApiURL+"/customers/vehicles/mappings", &res)

	return lo.Map(
		lo.Filter(res, func(v VehicleMapping, _ int) bool {
			return v.MappingType == "PRIMARY"
		}), func(v VehicleMapping, _ int) string {
			return v.Vin
		},
	), err
}

func (v *API) GetContainers() ([]Container, error) {
	var res struct {
		Containers []Container
	}
	if err := v.GetJSON(ApiURL+"/customers/containers", &res); err != nil {
		return nil, err
	}
	return res.Containers, nil
}

func (v *API) CreateContainer(data CreateContainer) (Container, error) {
	var res Container
	req, _ := request.New(http.MethodPost, ApiURL+"/customers/containers", request.MarshalJSON(data))
	err := v.DoJSON(req, &res)
	return res, err
}

func (v *API) DeleteContainer(id string) error {
	req, _ := request.New(http.MethodDelete, ApiURL+"/customers/containers/"+id, nil)
	var res any
	return v.DoJSON(req, &res)
}

func (v *API) GetTelematics(vin, container string) (ContainerContents, error) {
	var res ContainerContents
	uri := fmt.Sprintf(ApiURL+"/customers/vehicles/%s/telematicData?containerId=%s", vin, container)
	err := v.GetJSON(uri, &res)
	return res, err
}
