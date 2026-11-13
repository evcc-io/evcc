package mercedes

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	protos "github.com/evcc-io/evcc/vehicle/mercedes/pb"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/proto"
)

// API is an api.Vehicle implementation for Mercedes-Benz cars
type API struct {
	region   string
	log      *util.Logger
	identity *Identity
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
		Helper:   client,
		region:   identity.region,
		log:      log,
		identity: identity,
	}
}

func (v *API) Vehicles() ([]string, error) {
	var res VehiclesResponse

	url := fmt.Sprintf("%s/v2/vehicles", getBffUri(v.region))

	err := v.GetJSON(url, &res)
	if err != nil {
		return nil, err
	}

	all := res.AssignedVehicles
	for _, f := range res.Fleets {
		all = append(all, f.BookedVehicles...)
	}

	var vehicles []string
	for _, v := range all {
		if len(v.Vin) > 0 {
			vehicles = append(vehicles, v.Vin)
		} else {
			vehicles = append(vehicles, v.Fin)
		}
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

	return res, err
}

// Push returns the full update of the vehicle from the push api. It holds attributes the widget api no longer delivers.
func (v *API) Push(vin string) (*protos.VEPUpdate, error) {
	updates, err := v.identity.updatesG()
	if err != nil {
		return nil, err
	}

	update, ok := updates[vin]
	if !ok {
		return nil, api.ErrNotAvailable
	}

	return update, nil
}
