package charger

import (
	"context"
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

type Tessie struct {
	log                                 *util.Logger
	client                              *request.Helper
	Vin                                 string
	Location                            string
	Maxcurrent                          int64
	chargingStartedAfterLeavingGeofence bool
}

func init() {
	registry.AddCtx("tessie", NewTessieFromConfig)
}

func NewTessieFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Vin        string
		Token      string
		Location   string
		Maxcurrent int64
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("tessie")

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cc.Token})
	oauthClient := oauth2.NewClient(ctx, tokenSource)

	client := request.NewHelper(log)
	client.Client = oauthClient

	t := &Tessie{
		log:        log,
		client:     client,
		Vin:        cc.Vin,
		Location:   cc.Location,
		Maxcurrent: cc.Maxcurrent,
	}

	return t, nil
}

func (t *Tessie) Enabled() (bool, error) {
	locationName, err := t.getLocationName()
	if err != nil {
		return false, err
	}
	locationMatch := t.Location == locationName || t.Location == "always"

	if !locationMatch {
		return false, nil
	}

	url := fmt.Sprintf("https://api.tessie.com/%s/state?values=charge_state", t.Vin)
	req, err := request.New(http.MethodGet, url, nil, nil)
	if err != nil {
		return false, err
	}

	var res struct {
		ChargeState struct {
			ChargingState string `json:"charging_state"`
		} `json:"charge_state"`
	}

	if err := t.client.DoJSON(req, &res); err != nil {
		return false, err
	}

	return res.ChargeState.ChargingState == "Charging", nil
}

func (t *Tessie) Enable(enable bool) error {
	command := "stop_charging"
	if enable {
		command = "start_charging"
	}

	url := fmt.Sprintf("https://api.tessie.com/%s/command/%s?retry_duration=40&wait_for_completion=true", t.Vin, command)
	req, err := request.New(http.MethodPost, url, nil, nil)
	if err != nil {
		return err
	}
	_, err = t.client.Do(req)
	if err != nil {
		return err
	}

	if !enable {
		err = t.MaxCurrent(32)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Tessie) MaxCurrent(current int64) error {
	locationName, err := t.getLocationName()
	if err != nil {
		return err
	}
	locationMatch := t.Location == locationName || t.Location == "always"

	if !locationMatch {
		return nil
	}
	url := fmt.Sprintf("https://api.tessie.com/%s/command/set_charging_amps?retry_duration=40&wait_for_completion=true&amps=%d", t.Vin, current)
	req, err := request.New(http.MethodPost, url, nil, nil)
	if err != nil {
		return err
	}
	_, err = t.client.Do(req)
	return err
}

func (t *Tessie) Status() (api.ChargeStatus, error) {
	locationName, err := t.getLocationName()
	if err != nil {
		return api.StatusNone, err
	}

	url := fmt.Sprintf("https://api.tessie.com/%s/state?values=charge_state", t.Vin)
	req, err := request.New(http.MethodGet, url, nil, nil)
	if err != nil {
		return api.StatusNone, err
	}

	var res struct {
		ChargeState struct {
			ChargingState      string `json:"charging_state"`
			ChargePortDoorOpen bool   `json:"charge_port_door_open"`
		} `json:"charge_state"`
	}

	if err := t.client.DoJSON(req, &res); err != nil {
		return api.StatusNone, err
	}

	locationMatch := t.Location == locationName || t.Location == "always"

	if !locationMatch {
		if !t.chargingStartedAfterLeavingGeofence {
			if err := t.startCharging(); err != nil {
				t.log.ERROR.Printf("Failed to start charging after leaving geofence: %v", err)
			} else {
				t.chargingStartedAfterLeavingGeofence = true
			}
		}
		return api.StatusA, nil
	} else {
		t.chargingStartedAfterLeavingGeofence = false
	}

	switch res.ChargeState.ChargingState {
	case "Charging":
		return api.StatusC, nil
	case "Complete":
		return api.StatusA, nil
	default:
		if res.ChargeState.ChargePortDoorOpen {
			return api.StatusB, nil
		}
		return api.StatusA, nil
	}
}

func (t *Tessie) CurrentPower() (float64, error) {
	locationName, err := t.getLocationName()
	if err != nil {
		return 0, err
	}
	locationMatch := t.Location == locationName || t.Location == "always"

	if !locationMatch {
		return 0, nil
	}

	url := fmt.Sprintf("https://api.tessie.com/%s/state?values=charge_state", t.Vin)
	req, err := request.New(http.MethodGet, url, nil, nil)
	if err != nil {
		return 0, err
	}

	var res struct {
		ChargeState struct {
			ChargerPower float64 `json:"charger_power"`
		} `json:"charge_state"`
	}

	if err := t.client.DoJSON(req, &res); err != nil {
		return 0, err
	}

	return res.ChargeState.ChargerPower * 1000, nil
}

func (t *Tessie) ChargedEnergy() (float64, error) {
	locationName, err := t.getLocationName()
	if err != nil {
		return 0, err
	}
	locationMatch := t.Location == locationName || t.Location == "always"

	if !locationMatch {
		return 0, nil
	}

	url := fmt.Sprintf("https://api.tessie.com/%s/state?values=charge_state", t.Vin)
	req, err := request.New(http.MethodGet, url, nil, nil)
	if err != nil {
		return 0, err
	}

	var res struct {
		ChargeState struct {
			ChargeEnergyAdded float64 `json:"charge_energy_added"`
		} `json:"charge_state"`
	}

	if err := t.client.DoJSON(req, &res); err != nil {
		return 0, err
	}

	return res.ChargeState.ChargeEnergyAdded, nil
}

func (t *Tessie) startCharging() error {
	url := fmt.Sprintf("https://api.tessie.com/%s/command/start_charging?retry_duration=40&wait_for_completion=true", t.Vin)
	req, err := request.New(http.MethodPost, url, nil, nil)
	if err != nil {
		return err
	}
	_, err = t.client.Do(req)
	return err
}

func (t *Tessie) getLocationName() (string, error) {
	url := fmt.Sprintf("https://api.tessie.com/%s/location", t.Vin)
	req, err := request.New(http.MethodGet, url, nil, nil)
	if err != nil {
		return "", err
	}

	var res struct {
		Latitude      float64 `json:"latitude"`
		Longitude     float64 `json:"longitude"`
		Address       string  `json:"address"`
		SavedLocation string  `json:"saved_location"`
	}

	if err := t.client.DoJSON(req, &res); err != nil {
		return "", err
	}

	return res.SavedLocation, nil
}
