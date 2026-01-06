package subaru

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
	BaseUrl                  = "https://b2c-login.toyota-europe.com"
	ApiBaseUrl               = "https://ctpa-oneapi.tceu-ctp-prd.toyotaconnectedeurope.io"
	AccessTokenPath          = "oauth2/realms/root/realms/alliance-subaru/access_token"
	AuthenticationPath       = "json/realms/root/realms/alliance-subaru/authenticate?authIndexType=service&authIndexValue=oneapp"
	AuthorizationPath        = "oauth2/realms/root/realms/alliance-subaru/authorize?client_id=8c4921b0b08901fef389ce1af49c4e10.subaru.com&scope=openid+profile+write&response_type=code&redirect_uri=com.subaru.oneapp:/oauth2Callback&code_challenge=plain&code_challenge_method=plain"
	VehicleGuidPath          = "v2/vehicle/guid"
	RemoteElectricStatusPath = "v1/vehicle/electric/status"
	ApiKey                   = "tTZipv6liF74PwMfk9Ed68AQ0bISswwf3iHQdqcF"
	ClientRefKey             = "2.19.0"
)

type API struct {
	*request.Helper
	log       *util.Logger
	identity  *Identity
	clientRef string
}

func NewAPI(log *util.Logger, identity *Identity) *API {
	v := &API{
		Helper:   request.NewHelper(log),
		log:      log,
		identity: identity,
	}

	v.Timeout = 120 * time.Second

	v.Transport = &oauth2.Transport{
		Source: identity,
		Base:   v.Transport,
	}

	h := hmac.New(sha256.New, []byte(ClientRefKey))
	h.Write([]byte(v.identity.uuid))
	v.clientRef = hex.EncodeToString(h.Sum(nil))

	headers := map[string]string{
		"Accept":       request.JSONContent,
		"x-guid":       v.identity.uuid,
		"x-api-key":    ApiKey,
		"x-client-ref": v.clientRef,
		"x-appversion": ClientRefKey,
		"X-Appbrand":   "S",
	}

	v.Transport = &transport.Decorator{
		Decorator: transport.DecorateHeaders(headers),
		Base:      v.Transport,
	}

	return v
}

func (v *API) IDToken() string {
	return v.identity.IDToken()
}

func (v *API) Vehicles() ([]string, error) {
	uri := fmt.Sprintf("%s/%s", ApiBaseUrl, VehicleGuidPath)

	req, err := request.New(http.MethodGet, uri, nil, nil)
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
	uri := fmt.Sprintf("%s/%s?vin=%s", ApiBaseUrl, RemoteElectricStatusPath, vin)

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"vin": vin,
	})
	var status Status
	if err == nil {
		err = v.DoJSON(req, &status)
	}
	return status, err
}
