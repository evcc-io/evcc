package charger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/jq"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle"
	"github.com/evcc-io/evcc/vehicle/tesla"
	teslaclient "github.com/evcc-io/tesla-proxy-client"
	"github.com/itchyny/gojq"
	"golang.org/x/oauth2"
)

// twc3TeslaConfig is the optional tesla block enabling vehicle-independent on/off.
type twc3TeslaConfig struct {
	Credentials vehicle.ClientCredentials // ID (secret only needed for initial token mint, not for evcc's refresh)
	Tokens      vehicle.Tokens            // Access, Refresh
}

// twc3Fleet sends vehicle-independent on/off commands via the Tesla Fleet API
type twc3Fleet struct {
	client     *request.Helper // with oauth2 transport
	commandURL string          // fleetBaseURL + /api/1/energy_sites/{id}/command
	din        string
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

// newTwc3Fleet sets up the optional Tesla Fleet on/off control: it validates the
// tesla block, builds the authenticated fleet client, auto-discovers region, wall
// connector DIN and energy site, and resolves the on/off current (explicit override
// or the box's commissioned maximum). It returns the fleet and that current; a
// max-current read failure is non-fatal (current 0 -> on/off mode stays unavailable).
func newTwc3Fleet(log *util.Logger, local *request.Helper, baseURI string, cfg *twc3TeslaConfig, switchCurrentOverride float64) (*twc3Fleet, float64, error) {
	// the template renders the tesla block as soon as any field is set, so a
	// partial config surfaces here instead of being silently ignored
	if cfg.Credentials.ID == "" || cfg.Tokens.Access == "" || cfg.Tokens.Refresh == "" {
		return nil, 0, errors.New("tesla: clientId, accessToken and refreshToken are all required")
	}

	// redact the tesla secrets on the shared logger before any token handling
	log.Redact(
		cfg.Tokens.Access, cfg.Tokens.Refresh,
		cfg.Credentials.ID, cfg.Credentials.Secret,
	)

	// switchCurrent override only takes effect in this on/off fallback mode
	if switchCurrentOverride != 0 && (switchCurrentOverride < 1 || switchCurrentOverride > 32) {
		return nil, 0, fmt.Errorf("switchCurrent must be between 1 and 32 A, got %g", switchCurrentOverride)
	}

	token, err := cfg.Tokens.Token()
	if err != nil {
		return nil, 0, err
	}

	identity, err := tesla.NewIdentity(log, tesla.OAuth2Config(cfg.Credentials.ID, cfg.Credentials.Secret), token)
	if err != nil {
		return nil, 0, err
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
		return nil, 0, err
	}
	region, err := tc.UserRegion()
	if err != nil {
		return nil, 0, err
	}
	fleetBaseURL := strings.TrimRight(region.FleetApiBaseUrl, "/")

	// derive the wall connector DIN locally from this box (works per-uri, also with multiple TWC3s)
	din, err := localDIN(local, baseURI)
	if err != nil {
		return nil, 0, err
	}

	// find the energy site containing this wall connector
	energySiteID, err := discoverEnergySite(fleetClient, fleetBaseURL, din)
	if err != nil {
		return nil, 0, err
	}

	f := &twc3Fleet{
		client:     fleetClient,
		commandURL: fmt.Sprintf("%s/api/1/energy_sites/%d/command", fleetBaseURL, energySiteID),
		din:        din,
	}

	// determine the on/off current: explicit override, else the wall connector's
	// commissioned maximum (read once at startup - it does not change). On read
	// failure there is no fallback; the on/off mode stays unavailable (logged here).
	switchCurrent := switchCurrentOverride
	if switchCurrent == 0 {
		if a, err := f.maxOutputCurrent(); err != nil {
			log.WARN.Printf("could not read max output current, on/off mode disabled (set switchCurrent to enable): %v", err)
		} else {
			switchCurrent = a
		}
	}

	return f, switchCurrent, nil
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
