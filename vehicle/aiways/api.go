package aiways

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// https://github.com/snaptec/openWB/blob/master/modules/soc_aiways/aiways_get_soc.py

const URI = "https://coiapp-api-eu.ai-ways.com:10443"

// API implements the Aiways api
type API struct {
	*request.Helper
	user, hash string
}

// New creates a new Aiways API
func NewAPI(log *util.Logger, user, password string) *API {
	hash := md5.New()
	hash.Write([]byte(password))

	str := hex.EncodeToString(hash.Sum(nil))
	log.Redact(str)

	v := &API{
		Helper: request.NewHelper(log),
		user:   user,
		hash:   str,
	}

	v.Client.Transport = &transport.Decorator{
		Base:      v.Client.Transport,
		Decorator: transport.DecorateHeaders(map[string]string{
			// "apikey": EmobilityOAuth2Config.ClientID,
		}),
	}

	return v
}

type Vehicle struct {
	VIN, VehicleName, VehicleID string
}

func (v *API) Vehicles() ([]Vehicle, error) {
	var res User

	data := struct {
		Account  string `json:"account"`
		Password string `json:"password"`
	}{
		Account:  v.user,
		Password: v.hash,
	}

	uri := fmt.Sprintf("%s/aiways-passport-service/passport/login/password", URI)
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

	if err == nil {
		if err = v.DoJSON(req, &res); err == nil && res.Data == nil {
			err = errors.New(res.Message)
		}
	}

	return nil, err
}

func (v *API) Status(vin string) (StatusResponse, error) {
	var res StatusResponse

	data2 := struct {
		UserId int64  `json:"userId"`
		VIN    string `json:"vin"`
	}{
		UserId: 123,
		VIN:    vin,
	}

	uri := fmt.Sprintf("%s/app/vc/getCondition", URI)
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data2), map[string]string{
		"Content-Type": request.JSONContent,
		"Accept":       request.JSONContent,
		"Token":        "res.Data.Token",
	})

	if err == nil {
		if err = v.DoJSON(req, &res); err == nil && res.Data == nil {
			err = errors.New(res.Message)
		}
	}

	return res, err
}
