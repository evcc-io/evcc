package mercedes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	protos "github.com/evcc-io/evcc/vehicle/mercedes/pb"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/proto"
)

// API is an api.Vehicle implementation for Mercedes-Benz cars
type API struct {
	identity *Identity
	log      *util.Logger
	*request.Helper
}

type SetupAPI struct {
	log     *util.Logger
	account string
	region  string
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
		identity: identity,
		log:      log,
	}
}

func NewSetupAPI(log *util.Logger, account string, region string) *SetupAPI {
	client := request.NewHelper(log)

	client.Transport = &transport.Decorator{
		Base:      client.Transport,
		Decorator: transport.DecorateHeaders(mbheaders(true, region)),
	}

	return &SetupAPI{
		Helper:  client,
		log:     log,
		region:  region,
		account: account,
	}
}

func (v *API) Vehicles() ([]string, error) {
	var res VehiclesResponse

	url := fmt.Sprintf("%s/v2/vehicles", getBffUri(v.identity.region))

	err := v.GetJSON(url, &res)

	var vehicles []string
	for _, v := range res.assignedVehicles {
		vehicles = append(vehicles, v.fin)
	}

	return vehicles, err
}

func (v *API) Status(vin string) (StatusResponse, error) {
	var res StatusResponse

	uri := fmt.Sprintf("%s/v1/vehicle/%s/vehicleattributes", getWidgetUri(v.identity.region), vin)

	data, err := v.GetBody(uri)

	if err != nil {
		return res, err
	}

	message := &protos.VEPUpdate{}
	if err = proto.Unmarshal(data, message); err != nil {
		return res, err
	}

	if val, ok := message.Attributes["odo"]; ok {
		if val.GetNilValue() {
			res.VehicleInfo.Odometer.Value = 0
			res.VehicleInfo.Odometer.Unit = ""
		} else {
			res.VehicleInfo.Odometer.Value = int(val.GetIntValue())
			res.VehicleInfo.Odometer.Unit = val.GetDistanceUnit().String()
		}
	}

	if val, ok := message.Attributes["soc"]; ok {
		if val.GetNilValue() {
			res.EvInfo.Battery.StateOfCharge = 0
		} else {
			res.EvInfo.Battery.StateOfCharge = float64(val.GetIntValue())
		}
	}

	if val, ok := message.Attributes["rangeelectric"]; ok {
		if val.GetNilValue() {
			res.EvInfo.Battery.DistanceToEmpty.Value = 0
			res.EvInfo.Battery.DistanceToEmpty.Unit = ""
		} else {
			res.EvInfo.Battery.DistanceToEmpty.Value = int(val.GetIntValue())
			res.EvInfo.Battery.DistanceToEmpty.Unit = val.GetDistanceUnit().String()
		}
	}

	if val, ok := message.Attributes["endofchargetime"]; ok {
		if val.GetNilValue() {
			res.EvInfo.Battery.EndOfChargeTime = 0
		} else {
			res.EvInfo.Battery.EndOfChargeTime = int(val.GetIntValue())
		}
	}

	if val, ok := message.Attributes["chargingstatus"]; ok {
		if val.GetNilValue() {
			res.EvInfo.Battery.ChargingStatus = 3
		} else {
			res.EvInfo.Battery.ChargingStatus = int(val.GetIntValue())
		}
	} else {
		res.EvInfo.Battery.ChargingStatus = 3
	}

	return res, err
}

func (vs *SetupAPI) RequestPin() (bool, *string, error) {
	client := request.NewHelper(vs.log)

	client.Transport = &transport.Decorator{
		Base:      client.Transport,
		Decorator: transport.DecorateHeaders(mbheaders(false, vs.region)),
	}

	// Preflight request required to get a pin
	uri := fmt.Sprintf("%s/v1/config", getBffUri(vs.region))
	if _, err := vs.Helper.GetBody(uri); err != nil {
		return false, nil, err
	}

	uri = fmt.Sprintf("%s/v1/login", getBffUri(vs.region))
	nonce := uuid.New().String()
	data := fmt.Sprintf("{\"emailOrPhoneNumber\": \"%s\", \"countryCode\": \"EN\", \"nonce\": \"%s\"}", vs.account, nonce)
	res, err := client.Post(uri, "application/json", strings.NewReader(data))
	var pinResponse PinResponse
	if err != nil {
		return false, nil, err
	}

	defer res.Body.Close()
	var content []byte
	content, err = io.ReadAll(res.Body)
	if err != nil {
		return false, nil, err
	}
	json.Unmarshal(content, &pinResponse)

	// Only if the response field email is the same like the account an email is send by the servers.
	return pinResponse.UserName == vs.account, &nonce, err
}

func (vs *SetupAPI) RequestAccessToken(nonce string, pin string) (*oauth2.Token, error) {
	uri := fmt.Sprintf("%s/as/token.oauth2", IdUri)
	data := fmt.Sprintf("client_id=%s&grant_type=password&password=%s:%s&scope=openid email phone profile offline_access ciam-uid&username=%s", ClientId, nonce, pin, vs.account)

	req, err := request.New(http.MethodPost, uri, strings.NewReader(data), mbheaders(true, vs.region))

	var res MBToken
	if err == nil {
		err = vs.DoJSON(req, &res)
	}

	if err != nil {
		return nil, err
	}

	return res.GetToken(), err
}
