package vehicle

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/jlr"
)

// https://github.com/ardevd/jlrpy

// JLR is an api.Vehicle implementation for Jaguar LandRover cars
type JLR struct {
	*embed
	// *JLR.Provider
	*request.Helper
	user, password string
}

func init() {
	registry.Add("jaguar", NewJLRFromConfig)
	registry.Add("landrover", NewJLRFromConfig)
}

// NewJLRFromConfig creates a new vehicle
func NewJLRFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		DeviceID            string
		Expiry              time.Duration
		Cache               time.Duration
	}{
		Expiry: expiry,
		Cache:  interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	v := &JLR{
		embed: &cc.embed,
	}

	log := util.NewLogger("jlr").Redact(cc.User, cc.Password, cc.VIN, cc.DeviceID)

	// uid := uuid.New()
	// deviceId := uid.String()
	// fmt.Println("use this deviceid:", deviceId)
	cc.DeviceID = "d565375c-49a1-4b3d-93f6-79044033c414"

	identity := jlr.NewIdentity(log, cc.User, cc.Password, cc.DeviceID)

	err := identity.Login()
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	api := jlr.NewAPI(log, cc.DeviceID, identity)

	user, err := api.User(v.user)
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	cc.VIN, err = ensureVehicle(cc.VIN, func() ([]string, error) {
		return api.Vehicles(user.UserId)
	})

	// if err == nil {
	// 	v.Provider = jlr.NewProvider(api, cc.VIN, cc.Expiry, cc.Cache)
	// }

	return v, err
}

func (v *JLR) RegisterDevice(device string) error {
	var t jlr.Token

	data := map[string]string{
		"access_token":        t.AccessToken,
		"authorization_token": t.AuthToken,
		"expires_in":          "86400",
		"deviceID":            device}

	uri := fmt.Sprintf("%s/users/%s/clients", jlr.IFOP_BASE_URL, url.PathEscape(v.user))

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		err = v.DoJSON(req, nil)
	}

	return err
}
