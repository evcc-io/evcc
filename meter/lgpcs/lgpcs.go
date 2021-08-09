package lgpcs

// pcs ... the LG power conditioning system - converts the PV (or battery) - DC current into AC current (and controls the batteries)

import (
	"fmt"
	"sync"
	"strings"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// URIs
const (
	LoginURI   = "/v1/login"
	MeterURI   = "/v1/user/essinfo/home"
)

type LgEssData struct {
	GridPower 	float64	// < 0 if selling to grid
	GridEnergy 	float64	// total grid energy of this day in kWh
	PvPower		float64	// power of PV
	PvEnergy	float64	// total pv energy of this day in kWh
	BatPower	float64	// power of battery, < 0 if charging the battery
	BatSoC		float64	// battery state of charge
	lastRefreshTimeSec int64 // time of last refresh (unix time in seconds since jan 1st 1970 UTC)
}

type LgPcsCom struct {
	Helper		*request.Helper
	Log      	*util.Logger
	uri			string 		// IP address of the LG ESS Inverter - e.g. "https://192.168.1.28"
	password 	string		// registration number of the LG ESS Inverter - e.g. "DE2001..."
							// can also be found in the app - menuitem "Systeminformation"
	authKey 	string  	// auth_key returned during login and renewed with new login after expiration
	data		*LgEssData	// power/energy data
}

var once sync.Once
var instance *LgPcsCom

/**
 * Implements the singleton pattern to handel the access via the authkey to the PCS of the LGESSHome system
 */
func GetInstance(uri, password string) (*LgPcsCom,error) {

	var convertedUri string = ""

	// adapt uri if provided (!= "")
	if uri != "" {
		_, err := url.Parse(uri)
		if err != nil {
			return nil, fmt.Errorf("uri %s is invalid: %s", uri, err)
		}
		convertedUri = util.DefaultScheme(strings.TrimSuffix(uri, "/"), "https")
	}
	once.Do(func()  {
		log := util.NewLogger("lgess")
		instance = &LgPcsCom {
			Helper:   	request.NewHelper(log),
			Log:		log,
			uri:      	convertedUri,
			password: 	password,
			authKey:	"",
			data: 		&LgEssData {
							GridPower: 0.0,
							GridEnergy: 0.0,
							PvPower: 0.0,
							PvEnergy: 0.0,
							BatPower: 0.0,
							BatSoC: 0.0,
							lastRefreshTimeSec: 0,
						},
		}
		// ignore the self signed certificate
		instance.Helper.Client.Transport = request.NewTripper(log, request.InsecureTransport())
	})

	//instance.Log.DEBUG.Printf("Uri: [%v]\r\n", instance.uri)
	// It is suffucient to provide the uri once ... if not provided yet set uri now
	if instance.uri == "" {
		instance.uri = convertedUri
	}

	// Check if different uris are provided => report error
	if convertedUri != "" && instance.uri != convertedUri {
		return nil, fmt.Errorf("Uri mismatch for same lgess. Uri1: %s, Uri2: %s", instance.uri, convertedUri)
	}

	// It is sufficient to provide the password once ... if not provided yet set password now
	if instance.password == "" {
		instance.password = password
	}
	// Check if different passwors are provided => report error
	if password != "" && instance.password != password {
		return nil, fmt.Errorf("Password mismatch for same lgess. Pass1: %s, Pass2: %s", instance.password, password)
	}

	// Do first login if no authKey exists and uri and password exist
	if instance.authKey == "" && instance.uri != "" && instance.password != "" {
		instance.Log.DEBUG.Printf("Initial Login\r\n")
		if err := instance.Login(); err != nil {
			return nil, err
		}
	}

	return instance, nil
}

// Login calls login and stores the returned authorization key
func (m *LgPcsCom) Login() error {
	data := map[string]interface{}{
		"password": m.password,
	}

	m.Log.DEBUG.Printf("Login Uri: %v\r\n",m.uri)

	req, err := request.New(http.MethodPut, m.uri+LoginURI, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	// read auth_key from response body
	var result map[string]interface{}
	// use DoJSON as it will close the response body
	if err := m.Helper.DoJSON(req,&result); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// check login response status
	if status := result["status"].(string); status != "success" {
		return fmt.Errorf("login failed - status: %w", status)
	}
	// read auth_key from response
	m.authKey = result["auth_key"].(string)
	m.Log.DEBUG.Printf("Login success. AuthKey:%v\r\n",m.authKey)
	return nil
}

func (m *LgPcsCom) GetData() (*LgEssData,error) {
	err := m.ReadData()
	if err != nil {
		return nil, err
	}
	return m.data, nil
}

/*
 * Read data from lgess pcs if more than 2 seconds expired since last access.
 * Tries to re-login if "405" auth_key expired is returned
*/
func (m *LgPcsCom) ReadData() error {

	currentTimeSec:= time.Now().Unix()
	// refresh when at least 2 seconds elapsed since the last ReadData request.
	if (currentTimeSec - m.data.lastRefreshTimeSec) < 2 {
		return nil
	}
	m.data.lastRefreshTimeSec = currentTimeSec

	data := map[string]interface{}{
		"auth_key": m.authKey,
	}
	m.Log.DEBUG.Printf("Data refresh using auth_key: %v\r\n",m.authKey)
	req, err := request.New(http.MethodPost, m.uri+MeterURI, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	var res map[string]interface{}
	// re-login if request returns 405-error
	if err := m.Helper.DoJSON(req, &res); err != nil {
		m.Log.DEBUG.Printf("Data refresh failed with response: [%v]\r\n",err)
		// 405 if authKey expired - try re-login (only once)
		if strings.Contains(err.Error(), "405") {
			err = m.Login()
			if err != nil {
				m.Log.ERROR.Printf("Re-Login failed - error: [%v]\r\n",err)
				return err
			}
			data["auth_key"] = m.authKey
			req, err = request.New(http.MethodPost, m.uri+MeterURI, request.MarshalJSON(data), request.JSONEncoding)
			if err != nil {
				m.Log.ERROR.Printf("Failed to setup request - error: [%v]\r\n",err)
				return err
			}
			err = m.Helper.DoJSON(req, &res)
			if err != nil {
 				m.Log.ERROR.Printf("Re-Read data failed - error: [%v]\r\n",err)
				return err
			}
		} else {
			return err
		}
	}

	statistics := res["statistics"].(map[string]interface{})
	direction := res["direction"].(map[string]interface{})

	gridPower, err := strconv.ParseFloat(statistics["grid_power"].(string),64)
	if err != nil {
		return err
	}

	isGridSelling, err := strconv.Atoi(direction["is_grid_selling_"].(string))
	if err != nil {
		return err
	}

	// selling to grid: gridPower is negative, buying from grid: gridPower is positive
	if isGridSelling == 1 {
		gridPower = gridPower * (-1)
	}
	m.data.GridPower = gridPower

	pvPower, err := strconv.ParseFloat(statistics["pcs_pv_total_power"].(string),64)
	if err != nil {
		return err
	}
	m.data.PvPower = pvPower

	batPower, err := strconv.ParseFloat(statistics["batconv_power"].(string),64)
	if err != nil {
		return err
	}
	isBatDischarging, err := strconv.Atoi(direction["is_battery_discharging_"].(string))
	if err != nil {
		return err
	}

	// discharge battery: batPower is positive, charge battery: batPower is negative
	if isBatDischarging == 0 {
		batPower = batPower * (-1)
	}
	m.data.BatPower = batPower

	batSoC, err := strconv.ParseFloat(statistics["bat_user_soc"].(string),64)
	if err != nil {
		return err
	}
	m.data.BatSoC = batSoC

	gridEnergy, err := strconv.ParseFloat(statistics["current_grid_feed_in_energy"].(string),64)
	if err != nil {
		return err
	}
	// from Wh to kWh
	gridEnergy = gridEnergy / 1000
	m.data.GridEnergy = gridEnergy

	pvEnergy, err := strconv.ParseFloat(statistics["current_pv_generation_sum"].(string),64)
	if err != nil {
		return err
	}
	// from Wh to kWh
	pvEnergy = pvEnergy / 1000
	m.data.PvEnergy = pvEnergy
	return nil
}


/* example json result of uri: /v1/user/essinfo/home
{
    "statistics":
    {
        "pcs_pv_total_power": "0",
        "batconv_power": "1287",
        "bat_use": "1",
        "bat_status": "2",
        "bat_user_soc": "59.3",
        "load_power": "1289",
        "ac_output_power": "10",
        "load_today": "0.0",
        "grid_power": "2",
        "current_day_self_consumption": "70.7",
        "current_pv_generation_sum": "62253",
        "current_grid_feed_in_energy": "18250"
    },
    "direction":
    {
        "is_direct_consuming_": "0",
        "is_battery_charging_": "0",
        "is_battery_discharging_": "1",
        "is_grid_selling_": "0",
        "is_grid_buying_": "0",
        "is_charging_from_grid_": "0",
        "is_discharging_to_grid_": "0"
    },
    "operation":
    {
        "status": "start",
        "mode": "1",
        "pcs_standbymode": "false",
        "drm_mode0": "0",
        "remote_mode": "0",
        "drm_control": "255"
    },
    "wintermode":
    {
        "winter_status": "off",
        "backup_status": "off"
    },
    "backupmode": "",
    "pcs_fault":
    {
        "pcs_status": "pcs_ok",
        "pcs_op_status": "pcs_run"
    },
    "heatpump":
    {
        "heatpump_protocol": "0",
        "heatpump_activate": "off",
        "current_temp": "0",
        "heatpump_working": "off"
    },
    "evcharger":
    {
        "ev_activate": "off",
        "ev_power": "0"
    },
    "gridWaitingTime": "0"
}


{
    "statistics":
    {
        "pcs_pv_total_power": "638",
        "batconv_power": "469",
        "bat_use": "1",
        "bat_status": "0",
        "bat_user_soc": "55.7",
        "load_power": "703",
        "ac_output_power": "10",
        "load_today": "0.0",
        "grid_power": "404",
        "current_day_self_consumption": "94.6",
        "current_pv_generation_sum": "31386",
        "current_grid_feed_in_energy": "1697"
    },
    "direction":
    {
        "is_direct_consuming_": "1",
        "is_battery_charging_": "0",
        "is_battery_discharging_": "0",
        "is_grid_selling_": "1",
        "is_grid_buying_": "0",
        "is_charging_from_grid_": "0",
        "is_discharging_to_grid_": "0"
    },
    "operation":
    {
        "status": "start",
        "mode": "1",
        "pcs_standbymode": "false",
        "drm_mode0": "0",
        "remote_mode": "0",
        "drm_control": "255"
    },
    "wintermode":
    {
        "winter_status": "off",
        "backup_status": "off"
    },
    "backupmode": "",
    "pcs_fault":
    {
        "pcs_status": "pcs_ok",
        "pcs_op_status": "pcs_run"
    },
    "heatpump":
    {
        "heatpump_protocol": "0",
        "heatpump_activate": "off",
        "current_temp": "0",
        "heatpump_working": "off"
    },
    "evcharger":
    {
        "ev_activate": "off",
        "ev_power": "0"
    },
    "gridWaitingTime": "0"
}


*/
