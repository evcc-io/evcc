// Package lgpcs implements access to the LG pcs device (aka inverter).
// Pcs is the LG power conditioning system that converts the PV (or battery) - DC current into AC current (and controls the batteries)
package lgpcs

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// URIs
const (
	LoginURI     = "/v1/login"
	UserLoginURI = "/v1/user/setting/login"
	MeterURI     = "/v1/user/essinfo/home"
)

type MeterResponse struct {
	Statistics EssData
	Direction  struct {
		IsGridSelling        int `json:"is_grid_selling_,string"`
		IsBatteryDischarging int `json:"is_battery_discharging_,string"`
	}
}

type EssData struct {
	GridPower               float64 `json:"grid_power,string"`
	PvTotalPower            float64 `json:"pcs_pv_total_power,string"`
	BatConvPower            float64 `json:"batconv_power,string"`
	BatUserSoc              float64 `json:"bat_user_soc,string"`
	CurrentGridFeedInEnergy float64 `json:"current_grid_feed_in_energy,string"`
	CurrentPvGenerationSum  float64 `json:"current_pv_generation_sum,string"`
}

type Com struct {
	*request.Helper
	uri      string // URI address of the LG ESS inverter - e.g. "https://192.168.1.28"
	authPath string
	password string // registration number of the LG ESS Inverter - e.g. "DE2001..."
	authKey  string // auth_key returned during login and renewed with new login after expiration
	dataG    func() (EssData, error)
}

var (
	once     sync.Once
	instance *Com
)

// GetInstance implements the singleton pattern to handle the access via the authkey to the PCS of the LG ESS HOME system
func GetInstance(uri, registration, password string, cache time.Duration) (*Com, error) {
	uri = util.DefaultScheme(strings.TrimSuffix(uri, "/"), "https")

	var err error
	once.Do(func() {
		log := util.NewLogger("lgess")
		instance = &Com{
			Helper:   request.NewHelper(log),
			uri:      uri,
			authPath: UserLoginURI,
			password: password,
		}

		if registration != "" {
			instance.authPath = LoginURI
			instance.password = registration
		}

		// ignore the self signed certificate
		instance.Client.Transport = request.NewTripper(log, transport.Insecure())

		// caches the data access for the "cache" time duration
		// sends a new request to the pcs if the cache is expired and Data() requested
		instance.dataG = provider.Cached(instance.refreshData, cache)

		// do first login if no authKey exists and uri and password exist
		if instance.authKey == "" && instance.uri != "" && instance.password != "" {
			err = instance.Login()
		}
	})

	// check if different uris are provided
	if password != "" && registration != "" {
		return nil, errors.New("cannot have registration and password")
	}

	// check if different uris are provided
	if uri != "" && instance.uri != uri {
		return nil, fmt.Errorf("uri mismatch: %s vs %s", instance.uri, uri)
	}

	// check if different passwords are provided
	if password != "" && instance.password != password {
		return nil, errors.New("password mismatch")
	}

	return instance, err
}

// Login calls login and stores the returned authorization key
func (m *Com) Login() error {
	data := map[string]interface{}{
		"password": m.password,
	}

	req, err := request.New(http.MethodPut, m.uri+m.authPath, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	// read auth_key from response body
	var res struct {
		Status  string `json:"status,omitempty"`
		AuthKey string `json:"auth_key"`
	}

	if err := m.DoJSON(req, &res); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// check login response status
	if res.Status != "success" {
		return fmt.Errorf("login failed: %s", res.Status)
	}

	// read auth_key from response
	m.authKey = res.AuthKey

	return nil
}

// Data gives the data read from the pcs.
func (m *Com) Data() (EssData, error) {
	return m.dataG()
}

// refreshData reads data from lgess pcs. Tries to re-login if "405" auth_key expired is returned
func (m *Com) refreshData() (EssData, error) {
	data := map[string]interface{}{
		"auth_key": m.authKey,
	}

	req, err := request.New(http.MethodPost, m.uri+MeterURI, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return EssData{}, err
	}

	var resp MeterResponse

	if err = m.DoJSON(req, &resp); err != nil {
		// re-login if request returns 405-error
		if err2, ok := err.(request.StatusError); ok && err2.HasStatus(http.StatusMethodNotAllowed) {
			err = m.Login()

			if err == nil {
				data["auth_key"] = m.authKey
				req, err = request.New(http.MethodPost, m.uri+MeterURI, request.MarshalJSON(data), request.JSONEncoding)
			}

			if err == nil {
				err = m.DoJSON(req, &resp)
			}
		}
	}

	if err != nil {
		return EssData{}, err
	}

	res := resp.Statistics
	if resp.Direction.IsGridSelling > 0 {
		res.GridPower = -res.GridPower
	}

	// discharge battery: batPower is positive, charge battery: batPower is negative
	if resp.Direction.IsBatteryDischarging == 0 {
		res.BatConvPower = -res.BatConvPower
	}

	return res, nil
}
