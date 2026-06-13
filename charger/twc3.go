package charger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/jq"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle"
	"github.com/evcc-io/evcc/vehicle/tesla"
	teslaclient "github.com/evcc-io/tesla-proxy-client"
	"github.com/itchyny/gojq"
	"golang.org/x/oauth2"
)

// Twc3 is an api.Charger implementation for the Tesla Wall Connector Gen 3
type Twc3 struct {
	lp              loadpoint.API
	vitalsG         func() (Vitals, error)
	enabled         bool
	fleet           *twc3Fleet // nil without tesla block
	switchCurrent   float64    // on/off threshold current, only used in switch mode
	lockMaybeActive bool       // schedule may be gating the contactor (true = unknown/locked)
}

// twc3Fleet sends vehicle-independent on/off commands via the Tesla Fleet API
type twc3Fleet struct {
	client     *request.Helper // with oauth2 transport
	commandURL string          // fleetBaseURL + /api/1/energy_sites/{id}/command
	din        string
}

func init() {
	registry.Add("twc3", NewTwc3FromConfig)
}

// jq queries for the deeply-nested, non-contractual Fleet responses, compiled
// once - a bad query then panics at init instead of failing per request.
var (
	twc3ScheduleErrQuery   = jqMustParse(".. | .ConfigureChargeScheduleResponse? | .error? | numbers")
	twc3MaxOutputAmpsQuery = jqMustParse(".. | .max_output_current_amps? | numbers")
)

func jqMustParse(s string) *gojq.Query {
	q, err := gojq.Parse(s)
	if err != nil {
		panic(err)
	}
	return q
}

// Vitals is the /api/1/vitals response
type Vitals struct {
	ContactorClosed   bool    `json:"contactor_closed"`    // false
	VehicleConnected  bool    `json:"vehicle_connected"`   // false
	SessionS          int64   `json:"session_s"`           // 0
	GridV             float64 `json:"grid_v"`              // 230.1
	GridHz            float64 `json:"grid_hz"`             // 49.928
	VehicleCurrentA   float64 `json:"vehicle_current_a"`   // 0.1
	CurrentAA         float64 `json:"currentA_a"`          // 0.0
	CurrentBA         float64 `json:"currentB_a"`          // 0.1
	CurrentCA         float64 `json:"currentC_a"`          // 0.0
	CurrentNA         float64 `json:"currentN_a"`          // 0.0
	VoltageAV         float64 `json:"voltageA_v"`          // 0.0
	VoltageBV         float64 `json:"voltageB_v"`          // 0.0
	VoltageCV         float64 `json:"voltageC_v"`          // 0.0
	RelayCoilV        float64 `json:"relay_coil_v"`        // 11.8
	PcbaTempC         float64 `json:"pcba_temp_c"`         // 19.2
	HandleTempC       float64 `json:"handle_temp_c"`       // 15.3
	McuTempC          float64 `json:"mcu_temp_c"`          // 25.1
	UptimeS           int     `json:"uptime_s"`            // 831580
	InputThermopileUv float64 `json:"input_thermopile_uv"` //-233
	ProxV             float64 `json:"prox_v"`              // 0.0
	PilotHighV        float64 `json:"pilot_high_v"`        // 11.9
	PilotLowV         float64 `json:"pilot_low_v"`         // 11.9
	SessionEnergyWh   float64 `json:"session_energy_wh"`   // 22864.699
	ConfigStatus      int     `json:"config_status"`       // 5
	EvseState         int     `json:"evse_state"`          // 1
	CurrentAlerts     []any   `json:"current_alerts"`      // []
}

// NewTwc3FromConfig creates a new charger
func NewTwc3FromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		URI           string
		Cache         time.Duration
		SwitchCurrent float64 // on/off current override; 0 = read from wall connector
		Tesla         *struct {
			Credentials vehicle.ClientCredentials // ID (secret only needed for initial token mint, not for evcc's refresh)
			Tokens      vehicle.Tokens            // Access, Refresh
		}
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	c := &Twc3{
		lockMaybeActive: true, // unknown at startup -> first enable clears any stale lock once
	}

	log := util.NewLogger("twc3")
	client := request.NewHelper(log)
	baseURI := util.DefaultScheme(strings.TrimSuffix(cc.URI, "/"), "http")

	c.vitalsG = util.Cached(func() (Vitals, error) {
		var res Vitals
		err := client.GetJSON(baseURI+"/api/1/vitals", &res)
		return res, err
	}, cc.Cache)

	// optional vehicle-independent on/off via Tesla Fleet API
	if cc.Tesla != nil {
		// the template renders the tesla block as soon as any field is set, so a
		// partial config surfaces here instead of being silently ignored
		if cc.Tesla.Credentials.ID == "" || cc.Tesla.Tokens.Access == "" || cc.Tesla.Tokens.Refresh == "" {
			return nil, errors.New("tesla: clientId, accessToken and refreshToken are all required")
		}

		// switchCurrent override only takes effect in this on/off fallback mode
		if cc.SwitchCurrent != 0 && (cc.SwitchCurrent < 1 || cc.SwitchCurrent > 32) {
			return nil, fmt.Errorf("switchCurrent must be between 1 and 32 A, got %g", cc.SwitchCurrent)
		}

		token, err := cc.Tesla.Tokens.Token()
		if err != nil {
			return nil, err
		}

		log := util.NewLogger("twc3").Redact(
			cc.Tesla.Tokens.Access, cc.Tesla.Tokens.Refresh,
			cc.Tesla.Credentials.ID, cc.Tesla.Credentials.Secret,
		)

		identity, err := tesla.NewIdentity(log, tesla.OAuth2Config(cc.Tesla.Credentials.ID, cc.Tesla.Credentials.Secret), token)
		if err != nil {
			return nil, err
		}

		hc := request.NewClient(log)
		hc.Transport = &oauth2.Transport{
			Source: identity,
			Base:   hc.Transport,
		}
		fleetClient := &request.Helper{Client: hc}

		// fleet base url: auto-detect the account's region (same as the tesla vehicle)
		tc, err := teslaclient.NewClient(context.Background(), teslaclient.WithClient(hc))
		if err != nil {
			return nil, err
		}
		region, err := tc.UserRegion()
		if err != nil {
			return nil, err
		}
		fleetBaseURL := strings.TrimRight(region.FleetApiBaseUrl, "/")

		// derive the wall connector DIN locally from this box (works per-uri, also with multiple TWC3s)
		din, err := localDIN(client, baseURI)
		if err != nil {
			return nil, err
		}

		// find the energy site containing this wall connector
		energySiteID, err := discoverEnergySite(fleetClient, fleetBaseURL, din)
		if err != nil {
			return nil, err
		}

		c.fleet = &twc3Fleet{
			client:     fleetClient,
			commandURL: fmt.Sprintf("%s/api/1/energy_sites/%d/command", fleetBaseURL, energySiteID),
			din:        din,
		}

		// determine the on/off current: explicit override, else the wall connector's
		// commissioned maximum (read once at startup - it does not change). On read
		// failure there is no fallback; the on/off mode stays unavailable (logged below).
		if cc.SwitchCurrent != 0 {
			c.switchCurrent = cc.SwitchCurrent
		} else if a, err := c.fleet.maxOutputCurrent(); err != nil {
			log.WARN.Printf("could not read max output current, on/off mode disabled (set switchCurrent to enable): %v", err)
		} else {
			c.switchCurrent = a
		}
	}

	return c, nil
}

// localDIN derives the wall connector DIN (part_number--serial_number) from the
// local /api/1/version endpoint, so it never has to be configured manually.
func localDIN(client *request.Helper, baseURI string) (string, error) {
	var res struct {
		PartNumber   string `json:"part_number"`
		SerialNumber string `json:"serial_number"`
	}
	if err := client.GetJSON(baseURI+"/api/1/version", &res); err != nil {
		return "", fmt.Errorf("reading wall connector version: %w", err)
	}
	if res.PartNumber == "" || res.SerialNumber == "" {
		return "", errors.New("wall connector version: missing part/serial number")
	}
	return res.PartNumber + "--" + res.SerialNumber, nil
}

// discoverEnergySite returns the energy_site_id whose site_info lists the given
// wall connector DIN, so the site id never has to be configured manually.
func discoverEnergySite(client *request.Helper, fleetBaseURL, din string) (int64, error) {
	var products struct {
		Response []struct {
			EnergySiteID int64 `json:"energy_site_id"`
		} `json:"response"`
	}
	if err := client.GetJSON(fleetBaseURL+"/api/1/products", &products); err != nil {
		return 0, fmt.Errorf("listing energy sites: %w", err)
	}

	seen := make(map[int64]bool)
	for _, p := range products.Response {
		if p.EnergySiteID == 0 || seen[p.EnergySiteID] {
			continue
		}
		seen[p.EnergySiteID] = true

		var info struct {
			Response struct {
				Components struct {
					WallConnectors []struct {
						DIN string `json:"din"`
					} `json:"wall_connectors"`
				} `json:"components"`
			} `json:"response"`
		}
		if err := client.GetJSON(fmt.Sprintf("%s/api/1/energy_sites/%d/site_info", fleetBaseURL, p.EnergySiteID), &info); err != nil {
			continue // skip sites we cannot read
		}
		for _, wc := range info.Response.Components.WallConnectors {
			if wc.DIN == din {
				return p.EnergySiteID, nil
			}
		}
	}

	return 0, fmt.Errorf("wall connector %s not found in any energy site (token energy scope correct?)", din)
}

// utcTimeZone is the minimal time_zone the wall connector accepts (verified against
// real hardware): UTC with a single zero-offset transition. Sending only time_zone_id,
// or an empty transitions list, is rejected with error 4.
var utcTimeZone = json.RawMessage(`{"time_zone_id":"UTC","time_zone_info":{"transitions":[{"local_time_utc_offset":0,"timestamp":{"nanos":0,"seconds":0}}]}}`)

type wcTimePeriod struct {
	StartSeconds int `json:"start_seconds"`
	EndSeconds   int `json:"end_seconds"`
}

type wcDayTimePeriod struct {
	DayBitmask  int            `json:"day_bitmask"`
	TimePeriods []wcTimePeriod `json:"time_periods"`
}

// wcScheduleCmd is the configure_charge_schedule_request payload sent to the wall connector.
type wcScheduleCmd struct {
	CommandType       string `json:"command_type"`
	CommandProperties struct {
		Message struct {
			Wc struct {
				ConfigureChargeScheduleRequest struct {
					Config struct {
						Schedule struct {
							DayTimePeriods []wcDayTimePeriod `json:"day_time_periods"`
						} `json:"schedule"`
						Delay struct {
							MaxDelaySeconds int `json:"max_delay_seconds"`
						} `json:"delay"`
						EnableSchedule bool `json:"enable_schedule"`
					} `json:"config"`
					TimeZone json.RawMessage `json:"time_zone"`
				} `json:"configure_charge_schedule_request"`
			} `json:"wc"`
		} `json:"message"`
		IdentifierType int    `json:"identifier_type"`
		TargetID       string `json:"target_id"`
	} `json:"command_properties"`
}

// switchSchedule sends a configure_charge_schedule command: enable=true allows charging
// (schedule off), enable=false locks the contactor via an active past-window schedule.
func (f *twc3Fleet) switchSchedule(enable bool) error {
	var cmd wcScheduleCmd
	cmd.CommandType = "grpc_command"
	cmd.CommandProperties.TargetID = f.din
	cmd.CommandProperties.IdentifierType = 4

	ccsr := &cmd.CommandProperties.Message.Wc.ConfigureChargeScheduleRequest
	ccsr.TimeZone = utcTimeZone

	// charging blocked -> active schedule; charging allowed -> schedule disabled.
	ccsr.Config.EnableSchedule = !enable

	// The wall connector rejects an empty day_time_periods (error 4) even when the
	// schedule is disabled, so always send one valid window. Its only allowed slot is
	// the last elapsed half hour, so "now" is never inside it (relevant only while the
	// schedule is enabled). Computed in UTC to match the schedule's UTC time_zone, so
	// blocking works in any local time zone.
	s, e := offWindow(time.Now().UTC())
	ccsr.Config.Schedule.DayTimePeriods = []wcDayTimePeriod{{
		DayBitmask:  127,
		TimePeriods: []wcTimePeriod{{StartSeconds: s, EndSeconds: e}},
	}}

	req, err := request.New(http.MethodPost, f.commandURL, request.MarshalJSON(cmd), request.JSONEncoding)
	if err != nil {
		return err
	}

	body, err := f.client.DoBody(req)
	if err != nil {
		return err
	}

	// Success is signalled by ConfigureChargeScheduleResponse.error == 1 (same as the
	// Tesla app); anything else (e.g. 4) means rejected despite HTTP 200. The code sits
	// deep in a non-contractual envelope, so anchor on the unique response key with jq.
	if v, err := jq.Query(twc3ScheduleErrQuery, body); err == nil {
		if code, ok := v.(float64); ok && code == 1 {
			return nil
		}
		return fmt.Errorf("wall connector rejected schedule command (error %v)", v)
	}

	// no schedule response in the reply -> surface the top-level Fleet API error
	var res struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &res); err == nil && res.Error != "" {
		return errors.New(res.Error)
	}
	return errors.New("wall connector returned no schedule response")
}

// maxOutputCurrent reads the wall connector's commissioned maximum output current (A)
// via the get_config command - the same value the Tesla app shows under "max output current".
func (f *twc3Fleet) maxOutputCurrent() (float64, error) {
	var cmd struct {
		CommandType       string `json:"command_type"`
		CommandProperties struct {
			Message struct {
				Wc struct {
					GetConfigRequest struct{} `json:"get_config_request"`
				} `json:"wc"`
			} `json:"message"`
			IdentifierType int    `json:"identifier_type"`
			TargetID       string `json:"target_id"`
		} `json:"command_properties"`
	}
	cmd.CommandType = "grpc_command"
	cmd.CommandProperties.IdentifierType = 4
	cmd.CommandProperties.TargetID = f.din

	req, err := request.New(http.MethodPost, f.commandURL, request.MarshalJSON(cmd), request.JSONEncoding)
	if err != nil {
		return 0, err
	}

	body, err := f.client.DoBody(req)
	if err != nil {
		return 0, err
	}

	// the response is deeply nested in a non-contractual gRPC envelope; extract the
	// unique max_output_current_amps key with jq instead of mirroring the structure.
	v, err := jq.Query(twc3MaxOutputAmpsQuery, body)
	if err != nil {
		return 0, err
	}
	if a, ok := v.(float64); ok && a > 0 {
		return a, nil
	}
	return 0, errors.New("wall connector returned no max_output_current_amps")
}

// offWindow returns the last full, already-elapsed half hour as seconds-of-day
// (with midnight wrap), guaranteeing now is never inside [start,end). The caller
// passes UTC so the window matches the schedule's UTC time_zone.
func offWindow(now time.Time) (start, end int) {
	sec := now.Hour()*3600 + now.Minute()*60 + now.Second()
	end = (sec / 1800) * 1800 // start of the current half hour
	start = end - 1800
	if start < 0 { // before 00:30 -> wrap to 23:30-24:00
		start, end = 24*3600-1800, 24*3600
	}
	return
}

// switchMode reports whether the on/off fallback is in effect:
// fleet configured AND the connected vehicle cannot control current.
func (c *Twc3) switchMode() bool {
	return c.fleet != nil && c.lp != nil && !api.HasCap[api.CurrentController](c.lp.GetVehicle())
}

// Status implements the api.Charger interface
func (v *Twc3) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.vitalsG()
	switch {
	case res.ContactorClosed:
		status = api.StatusC
	case res.VehicleConnected:
		status = api.StatusB
	}

	return status, err
}

// Enabled implements the api.Charger interface
func (c *Twc3) Enabled() (bool, error) {
	return verifyEnabled(c, c.enabled)
}

// Enable implements the api.Charger interface
func (c *Twc3) Enable(enable bool) error {
	if c.lp == nil {
		return ErrLoadpointNotInitialized
	}

	// ignore disabling when the vehicle is already disconnected
	// https://github.com/evcc-io/evcc/issues/10213
	status, err := c.Status()
	if err != nil {
		return err
	}
	if status == api.StatusA && !enable {
		c.enabled = false
		return nil
	}

	if err := c.setCharging(enable); err != nil {
		return err
	}
	c.enabled = enable
	return nil
}

// setCharging starts/stops charging via the vehicle when it can, otherwise via the
// wall connector schedule (guest / non-Tesla / no vehicle).
func (c *Twc3) setCharging(enable bool) error {
	// 1. vehicle can start/stop (Tesla) -> control via vehicle
	if v, ok := api.Cap[api.ChargeController](c.lp.GetVehicle()); ok {
		// a guest may have locked the contactor via the schedule; clear it before
		// handing control back to the vehicle, but only when a lock may actually be
		// active - avoids a redundant Fleet API call on every enable. Disabling needs
		// no schedule change - the vehicle stops charging itself.
		if c.fleet != nil && enable && c.lockMaybeActive {
			if err := c.fleet.switchSchedule(true); err != nil {
				return err
			}
			c.lockMaybeActive = false
		}
		return v.ChargeEnable(enable)
	}

	// 2. fallback: vehicle cannot -> wall connector schedule
	if c.fleet != nil {
		if err := c.fleet.switchSchedule(enable); err != nil {
			return err
		}
		// the schedule now gates the contactor iff we disabled charging
		c.lockMaybeActive = !enable
		return nil
	}

	// 3. neither
	return errors.New("vehicle not capable of start/stop and no wall connector fallback configured")
}

// MaxCurrent implements the api.Charger interface
func (c *Twc3) MaxCurrent(current int64) error {
	if c.lp == nil {
		return ErrLoadpointNotInitialized
	}

	v, ok := api.Cap[api.CurrentController](c.lp.GetVehicle())
	if !ok {
		// hardware cannot set amps; on/off full load. Tolerate in all modes.
		return nil
	}

	return v.MaxCurrent(current)
}

var _ api.CurrentLimiter = (*Twc3)(nil)

// GetMinMaxCurrent implements the api.CurrentLimiter interface
func (c *Twc3) GetMinMaxCurrent() (float64, float64, error) {
	if c.switchMode() && c.switchCurrent > 0 {
		// min=max -> on/off only at full power
		return c.switchCurrent, c.switchCurrent, nil
	}
	// Tesla/classic, or unknown switch current: vehicle/loadpoint range applies
	return 0, 0, api.ErrNotAvailable
}

var _ api.CurrentGetter = (*Twc3)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (c *Twc3) GetMaxCurrent() (float64, error) {
	if c.lp == nil {
		return 0, ErrLoadpointNotInitialized
	}

	v, ok := api.Cap[api.CurrentGetter](c.lp.GetVehicle())
	if !ok {
		return 0, api.ErrNotAvailable
	}

	return v.GetMaxCurrent()
}

var _ api.ChargeRater = (*Twc3)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (v *Twc3) ChargedEnergy() (float64, error) {
	res, err := v.vitalsG()
	return res.SessionEnergyWh / 1e3, err
}

var _ api.ConnectionTimer = (*Twc3)(nil)

// ConnectionDuration implements the api.ConnectionTimer interface
func (v *Twc3) ConnectionDuration() (time.Duration, error) {
	res, err := v.vitalsG()
	return time.Duration(res.SessionS) * time.Second, err
}

// removed: https://github.com/evcc-io/evcc/issues/13555
// var _ api.ChargeTimer = (*Twc3)(nil)

// Use workaround if voltageC_v is approximately half of grid_v
//
//	"voltageA_v": 241.5,
//	"voltageB_v": 241.5,
//	"voltageC_v": 118.7,
//
// Default state is ~2V on all phases unless charging
func (v *Twc3) isSplitPhase(res Vitals) bool {
	return math.Abs(res.VoltageCV-res.GridV/2) < 25
}

var _ api.PhaseCurrents = (*Twc3)(nil)

// Currents implements the api.PhaseCurrents interface
func (v *Twc3) Currents() (float64, float64, float64, error) {
	res, err := v.vitalsG()
	if v.isSplitPhase(res) {
		return res.CurrentAA + res.CurrentBA, 0, 0, err
	}
	return res.CurrentAA, res.CurrentBA, res.CurrentCA, err
}

var _ api.Meter = (*Twc3)(nil)

// CurrentPower implements the api.Meter interface
func (v *Twc3) CurrentPower() (float64, error) {
	res, err := v.vitalsG()
	if res.ContactorClosed {
		if v.isSplitPhase(res) {
			return (res.CurrentAA * res.VoltageAV) + (res.CurrentBA * res.VoltageBV), err
		}
		return (res.CurrentAA * res.VoltageAV) + (res.CurrentBA * res.VoltageBV) + (res.CurrentCA * res.VoltageCV), err
	}
	return 0, err
}

var _ api.PhaseVoltages = (*Twc3)(nil)

// Voltages implements the api.PhaseVoltages interface
func (v *Twc3) Voltages() (float64, float64, float64, error) {
	res, err := v.vitalsG()
	if v.isSplitPhase(res) {
		return (res.VoltageAV + res.VoltageBV) / 2, 0, 0, err
	}
	return res.VoltageAV, res.VoltageBV, res.VoltageCV, err
}

var _ loadpoint.Controller = (*Twc3)(nil)

// LoadpointControl implements loadpoint.Controller
func (v *Twc3) LoadpointControl(lp loadpoint.API) {
	v.lp = lp
}
