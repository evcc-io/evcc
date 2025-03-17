package charger

import (
        "fmt"
        "net/http"

        "github.com/evcc-io/evcc/api"
        "github.com/evcc-io/evcc/util"
        "github.com/evcc-io/evcc/util/request"
)

type Tessie struct {
        log        *util.Logger
        client     *request.Helper
        vin        string
        token      string
        location   string
        maxcurrent int64
}

func init() {
        registry.Add("tessie", NewTessieFromConfig)
}

func NewTessieFromConfig(other map[string]interface{}) (api.Charger, error) {
        cc := struct {
                Vin        string
                Token      string
                Location   string
                maxcurrent int64
        }{}

        if err := util.DecodeOther(other, &cc); err != nil {
                return nil, err
        }
        log := util.NewLogger("tessie")

        client := request.NewHelper(log)

        t := &Tessie{
                log:        log,
                client:     client,
                vin:        cc.Vin,
                token:      cc.Token,
                location:   cc.Location,
                maxcurrent: cc.maxcurrent,
        }

        return t, nil
}

func (t *Tessie) Enabled() (bool, error) {
        locationName, err := t.getLocationName()
        if err != nil {
                return false, err
        }
        locationMatch := t.location == locationName || t.location == "always"

        if !locationMatch {
                return false, nil
        }

        url := fmt.Sprintf("https://api.tessie.com/%s/state?values=charge_state", t.vin)
        req, err := request.New(http.MethodGet, url, nil, map[string]string{
                "Authorization": fmt.Sprintf("Bearer %s", t.token),
        })
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

        url := fmt.Sprintf("https://api.tessie.com/%s/command/%s?retry_duration=40&wait_for_completion=true", t.vin, command)
        req, err := request.New(http.MethodPost, url, nil, map[string]string{
                "Authorization": fmt.Sprintf("Bearer %s", t.token),
        })
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
        locationMatch := t.location == locationName || t.location == "always"

        if !locationMatch {
                return nil
        }
        url := fmt.Sprintf("https://api.tessie.com/%s/command/set_charging_amps?retry_duration=40&wait_for_completion=true&amps=%d", t.vin, current)
        req, err := request.New(http.MethodPost, url, nil, map[string]string{
                "Authorization": fmt.Sprintf("Bearer %s", t.token),
        })
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

        url := fmt.Sprintf("https://api.tessie.com/%s/state?values=charge_state", t.vin)
        req, err := request.New(http.MethodGet, url, nil, map[string]string{
                "Authorization": fmt.Sprintf("Bearer %s", t.token),
        })
        if err != nil {
                return api.StatusNone, err
        }

        var res struct {
                ChargeState struct {
                        ChargingState        string `json:"charging_state"`
                        ChargePortDoorOpen   bool   `json:"charge_port_door_open"`
                } `json:"charge_state"`
        }

        if err := t.client.DoJSON(req, &res); err != nil {
                return api.StatusNone, err
        }

        locationMatch := t.location == locationName || t.location == "always"

        if !locationMatch {
                return api.StatusA, nil
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
        locationMatch := t.location == locationName || t.location == "always"

        if !locationMatch {
                return 0, nil
        }

        url := fmt.Sprintf("https://api.tessie.com/%s/state?values=charge_state", t.vin)
        req, err := request.New(http.MethodGet, url, nil, map[string]string{
                "Authorization": fmt.Sprintf("Bearer %s", t.token),
        })
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
        locationMatch := t.location == locationName || t.location == "always"

        if !locationMatch {
                return 0, nil
        }

        url := fmt.Sprintf("https://api.tessie.com/%s/state?values=charge_state", t.vin)
        req, err := request.New(http.MethodGet, url, nil, map[string]string{
                "Authorization": fmt.Sprintf("Bearer %s", t.token),
        })
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

func (t *Tessie) getLocationName() (string, error) {
        url := fmt.Sprintf("https://api.tessie.com/%s/location", t.vin)
        req, err := request.New(http.MethodGet, url, nil, map[string]string{
                "Authorization": fmt.Sprintf("Bearer %s", t.token),
        })
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
