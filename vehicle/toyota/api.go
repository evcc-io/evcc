package toyota

import (
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

const (
	BaseUrl                  = "https://oneapp:oneapp@b2c-login.toyota-europe.com"
	ApiBaseUrl               = "https://ctpa-oneapi.tceu-ctp-prd.toyotaconnectedeurope.io"
	AccessTokenPath          = "oauth2/realms/root/realms/tme/access_token"
	AuthenticationPath       = "json/realms/root/realms/tme/authenticate?authIndexType=service&authIndexValue=oneapp"
	AuthorizationPath        = "oauth2/realms/root/realms/tme/authorize?client_id=oneapp&scope=openid+profile+write&response_type=code&redirect_uri=com.toyota.oneapp:/oauth2Callback&code_challenge=plain&code_challenge_method=plain"
	VehicleGuidPath          = "v2/vehicle/guid"
	RemoteElectricStatusPath = "v1/global/remote/electric/status"
	ApiKey                   = "tTZipv6liF74PwMfk9Ed68AQ0bISswwf3iHQdqcF"
)

type API struct {
	*request.Helper
	log      *util.Logger
	identity *Identity
}

func NewAPI(log *util.Logger, identity *Identity) *API {
	v := &API{
		Helper:   request.NewHelper(log),
		log:      log,
		identity: identity,
	}

	v.Timeout = 120 * time.Second

	// replace client transport with authenticated transport
	v.Transport = &oauth2.Transport{
		Source: identity,
		Base:   v.Transport,
	}

	return v
}

func (v *API) Vehicles() ([]string, error) {
	uri := fmt.Sprintf("%s/%s", ApiBaseUrl, VehicleGuidPath)

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept":    request.JSONContent,
		"x-guid":    v.identity.uuid,
		"x-api-key": ApiKey,
	})
	var resp Vehicles
	if err == nil {
		err = v.DoJSON(req, &resp)
	}
	var vehicles []string
	for _, v := range resp.Payload {
		vehicles = append(vehicles, v.VIN)
	}
	return vehicles, err
}

func (v *API) Status(vin string) (Status, error) {
	uri := fmt.Sprintf("%s/%s", ApiBaseUrl, RemoteElectricStatusPath)

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept":    request.JSONContent,
		"x-guid":    v.identity.uuid,
		"x-api-key": ApiKey,
		"vin":       vin,
	})
	var status Status
	if err == nil {
		err = v.DoJSON(req, &status)
	}
	return status, err
}
