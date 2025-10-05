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

var Config = oauth2.Config{
	Scopes: []string{"authenticate_user", "openid", "cardata:streaming:read", "cardata:api:read"},
	Endpoint: oauth2.Endpoint{
		DeviceAuthURL: "https://customer.bmwgroup.com/gcdm/oauth/device/code",
		TokenURL:      "https://customer.bmwgroup.com/gcdm/oauth/token",
	},
}

// requiredKeys are the necessary data dictionary entities according to
// https://mybmwweb-utilities.api.bmw/de-de/utilities/bmw/api/cd/catalogue/file
var requiredKeys = []string{
	"vehicle.body.chargingPort.status",
	"vehicle.cabin.hvac.preconditioning.status.comfortState",
	"vehicle.drivetrain.batteryManagement.header",
	"vehicle.drivetrain.electricEngine.charging.hvStatus",
	"vehicle.drivetrain.electricEngine.charging.level",
	"vehicle.drivetrain.electricEngine.charging.timeToFullyCharged",
	"vehicle.drivetrain.electricEngine.kombiRemainingElectricRange",
	"vehicle.powertrain.electric.battery.stateOfCharge.target",
	"vehicle.vehicle.travelledDistance",
}

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
	return lo.Filter(res.Containers, func(c Container, _ int) bool {
		return c.Name == "evcc.io"
	}), nil
}

func (v *API) CreateContainer() error {
	data := CreateContainer{
		Name:                 "evcc.io",
		Purpose:              "evcc.io",
		TechnicalDescriptors: requiredKeys,
	}

	var res any
	req, _ := request.New(http.MethodPost, ApiURL+"/customers/containers", request.MarshalJSON(data))
	return v.DoJSON(req, &res)
}

// func (v *API) DeleteContainer() error {
// if *deleteContainer && len(containers) == 1 {
// 	req, _ := request.New(http.MethodDelete, apiUrl+"/customers/containers/"+containers[0].ContainerId, nil)

// 	var res any
// 	if err := client.DoJSON(req, &res); err != nil {
// 		return err
// 	}

// 	containers = nil
// }
// }

func (v *API) EnsureContainer() (string, error) {
	containers, err := v.GetContainers()
	if err != nil {
		return "", err
	}

	if len(containers) > 0 {
		return containers[0].ContainerId, nil
	}

	data := CreateContainer{
		Name:    "evcc.io",
		Purpose: "evcc.io",
		TechnicalDescriptors: []string{
			// https://mybmwweb-utilities.api.bmw/de-de/utilities/bmw/api/cd/catalogue/file
			"vehicle.body.chargingPort.status",
			"vehicle.cabin.hvac.preconditioning.status.comfortState",
			"vehicle.drivetrain.batteryManagement.header",
			"vehicle.drivetrain.electricEngine.charging.hvStatus",
			"vehicle.drivetrain.electricEngine.charging.level",
			"vehicle.drivetrain.electricEngine.charging.timeToFullyCharged",
			"vehicle.drivetrain.electricEngine.kombiRemainingElectricRange",
			"vehicle.powertrain.electric.battery.stateOfCharge.target",
			"vehicle.vehicle.travelledDistance",
		},
	}

	var res Container

	req, _ := request.New(http.MethodPost, ApiURL+"/customers/containers", request.MarshalJSON(data))
	err = v.DoJSON(req, &res)

	return res.ContainerId, err
}

func (v *API) GetTelematics(vin, container string) (TelematicData, error) {
	var res TelematicData
	uri := fmt.Sprintf(ApiURL+"/customers/vehicles/%s/telematicData?containerId=%s", vin, container)
	err := v.GetJSON(uri, &res)
	return res, err
}
