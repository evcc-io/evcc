package vehicle

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/silence"
	"github.com/thoas/go-funk"
	"golang.org/x/oauth2"
	"google.golang.org/api/googleapi"
	identitytoolkit "google.golang.org/api/identitytoolkit/v3"
	"google.golang.org/api/option"
)

// Silence is an api.Vehicle implementation for Silence S01 vehicles
type Silence struct {
	*embed
	*request.Helper
	vin  string
	apiG func() (interface{}, error)
}

func init() {
	registry.Add("silence", NewSilenceFromConfig)
}

// NewFordFromConfig creates a new vehicle
func NewSilenceFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed          `mapstructure:",squash"`
		User, Password string
		VIN            string
		Cache          time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, errors.New("missing user or password")
	}

	log := util.NewLogger("s01").Redact(cc.User, cc.Password)
	helper := request.NewHelper(log)

	ctx := context.Background()
	identitytoolkitService, err := identitytoolkit.NewService(ctx, option.WithHTTPClient(helper.Client))
	if err != nil {
		return nil, err
	}

	pwdReq := &identitytoolkit.IdentitytoolkitRelyingpartyVerifyPasswordRequest{
		Email:             cc.User,
		Password:          cc.Password,
		ReturnSecureToken: true,
	}

	call := identitytoolkitService.Relyingparty.VerifyPassword(pwdReq)

	pwdResp, err := call.Do(googleapi.QueryParameter("key", "AIzaSyCQYZCPvfl-y5QmzRrbUrCwR0RVNbyKqwI"))
	if err != nil {
		return nil, err
	}

	v := &Silence{
		embed:  &cc.embed,
		Helper: helper,
	}

	// TODO token refresh
	ts := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken:  pwdResp.IdToken,
		RefreshToken: pwdResp.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(pwdResp.ExpiresIn) * time.Second),
	})

	v.Client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   v.Client.Transport,
	}

	v.vin, err = ensureVehicle(strings.ToLower(cc.VIN), v.Vehicles)

	v.apiG = provider.NewCached(v.api, cc.Cache).InterfaceGetter()

	return v, err
}

// vehicles provides list of vehicles response
func (v *Silence) vehicles() ([]silence.Vehicle, error) {
	var resp []silence.Vehicle
	err := v.GetJSON(silence.ApiUri, &resp)
	return resp, err
}

// api provides the vehicle api response
func (v *Silence) api() (interface{}, error) {
	resp, err := v.vehicles()

	if err == nil {
		for _, vv := range resp {
			if vv.FrameNo == v.vin {
				return vv, nil
			}
		}

		err = errors.New("vehicle not found")
	}

	return nil, err
}

// Vehicles provides list of Vehicle IDs
func (v *Silence) Vehicles() ([]string, error) {
	var resp []silence.Vehicle

	err := v.GetJSON(silence.ApiUri, &resp)
	if err == nil {
		return funk.Map(resp, func(v silence.Vehicle) string {
			return v.FrameNo
		}).([]string), nil
	}

	return nil, err
}

// SoC implements the api.Vehicle interface
func (v *Silence) SoC() (float64, error) {
	res, err := v.apiG()

	if res, ok := res.(silence.Vehicle); err == nil && ok {
		return float64(res.BatterySoc), nil
	}

	return 0, err
}

var _ api.VehicleRange = (*Silence)(nil)

// Range implements the api.VehicleRange interface
func (v *Silence) Range() (int64, error) {
	res, err := v.apiG()

	if res, ok := res.(silence.Vehicle); err == nil && ok {
		return int64(res.Range), nil
	}

	return 0, err
}

var _ api.VehicleOdometer = (*Silence)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Silence) Odometer() (float64, error) {
	res, err := v.apiG()

	if res, ok := res.(silence.Vehicle); err == nil && ok {
		return float64(res.Odometer), nil
	}

	return 0, err
}
