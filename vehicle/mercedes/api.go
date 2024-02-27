package mercedes

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	protos "github.com/evcc-io/evcc/vehicle/mercedes/pb"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/proto"
)

// API is an api.Vehicle implementation for Mercedes-Benz cars
type API struct {
	region string
	log    *util.Logger
	*request.Helper
}

func NewAPI(log *util.Logger, identity *Identity) *API {
	client := request.NewHelper(log)

	client.Transport = &transport.Decorator{
		Base: &oauth2.Transport{
			Source: identity,
			Base:   client.Transport,
		},
		Decorator: transport.DecorateHeaders(mbheaders(false, identity.region)),
	}

	return &API{
		Helper: client,
		region: identity.region,
		log:    log,
	}
}

func (v *API) Vehicles() ([]string, error) {
	var res VehiclesResponse

	url := fmt.Sprintf("%s/v2/vehicles", getBffUri(v.region))

	err := v.GetJSON(url, &res)
	if err != nil {
		return nil, err
	}

	var vehicles []string
	for _, v := range res.AssignedVehicles {
		vehicles = append(vehicles, v.Fin)
	}

	return vehicles, err
}

func (v *API) Status(vin string) (StatusResponse, error) {
	var res StatusResponse

	uri := fmt.Sprintf("%s/v1/vehicle/%s/vehicleattributes", getWidgetUri(v.region), vin)

	data, err := v.GetBody(uri)
	if err != nil {
		return res, err
	}

	var message protos.VEPUpdate
	if err = proto.Unmarshal(data, &message); err != nil {
		return res, err
	}

	if val, ok := message.Attributes["odo"]; ok {
		res.VehicleInfo.Odometer.Value = int(val.GetIntValue())
		res.VehicleInfo.Odometer.Unit = val.GetDistanceUnit().String()
	}

	if val, ok := message.Attributes["soc"]; ok {
		res.EvInfo.Battery.StateOfCharge = float64(val.GetIntValue())
	}

	if val, ok := message.Attributes["rangeelectric"]; ok {
		res.EvInfo.Battery.DistanceToEmpty.Value = int(val.GetIntValue())
		res.EvInfo.Battery.DistanceToEmpty.Unit = val.GetDistanceUnit().String()
	}

	if val, ok := message.Attributes["endofchargetime"]; ok {
		res.EvInfo.Battery.EndOfChargeTime = int(val.GetIntValue())
	}

	if val, ok := message.Attributes["chargingstatus"]; ok {
		res.EvInfo.Battery.ChargingStatus = int(val.GetIntValue())
	} else {
		res.EvInfo.Battery.ChargingStatus = 3
	}

	return res, err
}
