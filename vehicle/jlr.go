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
	"github.com/google/uuid"
)

// https://github.com/ardevd/jlrpy

// JLR is an api.Vehicle implementation for Jaguar LandRover cars
type JLR struct {
	*embed
	*jlr.Provider
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

	if cc.DeviceID == "" {
		uid := uuid.New()
		cc.DeviceID = uid.String()
		log.WARN.Println("new device id generated, add `deviceid` to config:", cc.DeviceID)
	}

	identity := jlr.NewIdentity(log, cc.User, cc.Password, cc.DeviceID)

	token, err := identity.Login()
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	if err := v.RegisterDevice(log, cc.User, cc.DeviceID, token); err != nil {
		return nil, fmt.Errorf("device registry failed: %w", err)
	}

	api := jlr.NewAPI(log, cc.DeviceID, identity)

	user, err := api.User(cc.User)
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	cc.VIN, err = ensureVehicle(cc.VIN, func() ([]string, error) {
		return api.Vehicles(user.UserId)
	})

	if err == nil {
		v.Provider = jlr.NewProvider(api, cc.VIN, user.UserId, cc.Cache)
	}

	return v, err
}

func (v *JLR) RegisterDevice(log *util.Logger, user, device string, t jlr.Token) error {
	c := request.NewHelper(log)

	data := map[string]string{
		"access_token":        t.AccessToken,
		"authorization_token": t.AuthToken,
		"expires_in":          "86400",
		"deviceID":            device,
	}

	uri := fmt.Sprintf("%s/users/%s/clients", jlr.IFOP_BASE_URL, url.PathEscape(user))

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), map[string]string{
		"Authorization":           "Bearer " + t.AccessToken,
		"Content-type":            "application/json",
		"Accept":                  "application/json",
		"X-Device-Id":             device,
		"x-telematicsprogramtype": "jlrpy",
		"x-App-Id":                "ICR_JAGUAR",
		"x-App-Secret":            "018dd168-6271-707f-9fd4-aed2bf76905e",
	})
	if err == nil {
		_, err = c.DoBody(req)
	}

	return err
}
