package toyota

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

const (
	BaseUrl                          = "https://oneapp:oneapp@b2c-login.toyota-europe.com"
	ApiBaseUrl                       = "https://ctpa-oneapi.tceu-ctp-prd.toyotaconnectedeurope.io"
	AccessTokenPath                  = "oauth2/realms/root/realms/tme/access_token"
	AuthenticationPath               = "json/realms/root/realms/tme/authenticate?authIndexType=service&authIndexValue=oneapp"
	AuthorizationPath                = "oauth2/realms/root/realms/tme/authorize?client_id=oneapp&scope=openid+profile+write&response_type=code&redirect_uri=com.toyota.oneapp:/oauth2Callback&code_challenge=plain&code_challenge_method=plain"
	VehicleGuidPath                  = "v2/vehicle/guid"
	RemoteElectricStatusPath         = "v1/global/remote/electric/status"
	RemoteElectricRealtimeStatusPath = "v1/global/remote/electric/realtime-status"
	apiKey                           = "tTZipv6liF74PwMfk9Ed68AQ0bISswwf3iHQdqcF"
	clientRefKey                     = "3e0b15f6c9c87fbd"
	channel                          = "ONEAPP" // Required x-channel header value for Toyota OneApp API
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

	// create HMAC digest for x-client-ref header
	h := hmac.New(sha256.New, []byte(clientRefKey))
	h.Write([]byte(v.identity.uuid))
	clientRef := hex.EncodeToString(h.Sum(nil))

	// replace client transport with authenticated transport
	v.Transport = &transport.Decorator{
		Decorator: transport.DecorateHeaders(map[string]string{
			"Accept":       request.JSONContent,
			"x-guid":       v.identity.uuid,
			"x-api-key":    apiKey,
			"x-client-ref": clientRef,
			"x-appversion": clientRefKey,
			"x-channel":    channel,
		}),
		Base: &oauth2.Transport{
			Source: identity,
			Base:   v.Transport,
		},
	}

	return v
}

func (v *API) Vehicles() ([]string, error) {
	uri := fmt.Sprintf("%s/%s", ApiBaseUrl, VehicleGuidPath)

	var res Vehicles
	err := v.GetJSON(uri, &res)

	var vehicles []string
	for _, v := range res.Payload {
		vehicles = append(vehicles, v.VIN)
	}
	return vehicles, err
}

func (v *API) Status(vin string) (Status, error) {
	uri := fmt.Sprintf("%s/%s", ApiBaseUrl, RemoteElectricStatusPath)

	req, _ := request.New(http.MethodGet, uri, nil, map[string]string{
		"vin": vin,
	})

	var res Status
	err := v.DoJSON(req, &res)

	return res, err
}

func (v *API) RefreshStatus(vin string) error {
	uri := fmt.Sprintf("%s/%s", ApiBaseUrl, RemoteElectricRealtimeStatusPath)

	req, _ := request.New(http.MethodPost, uri, nil, map[string]string{
		"vin": vin,
	})

	_, err := v.DoBody(req)
	return err
}
