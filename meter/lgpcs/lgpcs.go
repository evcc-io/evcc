package lgpcs

// pcs ... the LG power conditioning system - converts the PV (or battery) - DC current into AC current (and controls the batteries)

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// URIs
const (
	LoginURI = "/v1/login"
	MeterURI = "/v1/user/essinfo/home"
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
	Helper   *request.Helper
	Log      *util.Logger
	uri      string // IP address of the LG ESS Inverter - e.g. "https://192.168.1.28"
	password string // registration number of the LG ESS Inverter - e.g. "DE2001..."
	authKey  string // auth_key returned during login and renewed with new login after expiration
}

var once sync.Once
var instance *Com

// GetInstance implements the singleton pattern to handel the access via the authkey to the PCS of the LGESSHome system
func GetInstance(uri, password string) (*Com, error) {
	uri = util.DefaultScheme(strings.TrimSuffix(uri, "/"), "https")

	once.Do(func() {
		log := util.NewLogger("lgess")
		instance = &Com{
			Helper:   request.NewHelper(log),
			Log:      log,
			uri:      uri,
			password: password,
		}

		// ignore the self signed certificate
		instance.Helper.Client.Transport = request.NewTripper(log, request.InsecureTransport())
	})

	// it is suffucient to provide the uri once ... if not provided yet set uri now
	if instance.uri == "" {
		instance.uri = uri
	}

	// check if different uris are provided
	if uri != "" && instance.uri != uri {
		return nil, fmt.Errorf("Uri mismatch for same lgess. Uri1: %s, Uri2: %s", instance.uri, uri)
	}

	// it is sufficient to provide the password once ... if not provided yet set password now
	if instance.password == "" {
		instance.password = password
	}

	// check if different passwords are provided
	if password != "" && instance.password != password {
		return nil, fmt.Errorf("Password mismatch for same lgess. Pass1: %s, Pass2: %s", instance.password, password)
	}

	// do first login if no authKey exists and uri and password exist
	if instance.authKey == "" && instance.uri != "" && instance.password != "" {
		instance.Log.DEBUG.Printf("Initial Login\r\n")
		if err := instance.Login(); err != nil {
			return nil, err
		}
	}

	return instance, nil
}

// login calls login and stores the returned authorization key
func (m *Com) Login() error {
	data := map[string]interface{}{
		"password": m.password,
	}

	m.Log.DEBUG.Printf("Login Uri: %v\r\n", m.uri)

	req, err := request.New(http.MethodPut, m.uri+LoginURI, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	// read auth_key from response body
	var res struct {
		Status  string
		AuthKey string
	}

	// use DoJSON as it will close the response body
	if err := m.Helper.DoJSON(req, &res); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// check login response status
	if res.Status != "success" {
		return fmt.Errorf("login failed - status: %s", res.Status)
	}

	// read auth_key from response
	m.authKey = res.AuthKey

	return nil
}

// Data reads data from lgess pcs. Tries to re-login if "405" auth_key expired is returned
func (m *Com) Data() (EssData, error) {
	data := map[string]interface{}{
		"auth_key": m.authKey,
	}

	req, err := request.New(http.MethodPost, m.uri+MeterURI, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return EssData{}, err
	}

	var resp MeterResponse

	// re-login if request returns 405-error
	if err := m.Helper.DoJSON(req, &resp); err != nil {
		m.Log.DEBUG.Printf("Data refresh failed with response: [%v]\r\n", err)
		// 405 if authKey expired - try re-login (only once)
		if strings.Contains(err.Error(), "405") {
			err = m.Login()
			if err != nil {
				m.Log.ERROR.Printf("Re-Login failed - error: [%v]\r\n", err)
				return EssData{}, err
			}

			data["auth_key"] = m.authKey

			req, err = request.New(http.MethodPost, m.uri+MeterURI, request.MarshalJSON(data), request.JSONEncoding)
			if err != nil {
				m.Log.ERROR.Printf("Failed to setup request - error: [%v]\r\n", err)
				return EssData{}, err
			}

			if err = m.Helper.DoJSON(req, &resp); err != nil {
				m.Log.ERROR.Printf("Re-Read data failed - error: [%v]\r\n", err)
				return EssData{}, err
			}
		} else {
			return EssData{}, err
		}
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
