package charger

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/charger/ghostone"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

// GhostEEBus charger implementation combining EEBus protocol for EV communication
// with Ghost platform REST API for phase switching and RFID identification.
type GhostEEBus struct {
	*EEBus
	*request.Helper
	uri     string // REST API base URL, e.g. "https://10.0.1.30/api/v2"
	hasRFID bool
}

func init() {
	registry.AddCtx("ghosteebus", NewGhostEEBusFromConfig)
}

// NewGhostEEBusFromConfig creates a GhostEEBus charger from generic config
func NewGhostEEBusFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	var cc struct {
		Ski           string
		Ip            string
		Meter         bool
		ChargedEnergy *bool
		User          string
		Password      string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// default true
	hasChargedEnergy := cc.ChargedEnergy == nil || *cc.ChargedEnergy

	return NewGhostEEBus(ctx, cc.Ski, cc.Ip, cc.User, cc.Password, cc.Meter, hasChargedEnergy)
}

// NewGhostEEBus creates a GhostEEBus charger combining EEBus with Ghost REST API
func NewGhostEEBus(ctx context.Context, ski, ip, user, password string, hasMeter, hasChargedEnergy bool) (api.Charger, error) {
	eb, err := newEEBus(ctx, ski, ip)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("ghost-eebus").Redact(user, password)

	uri := "https://" + ip + "/api/v2"

	wb := &GhostEEBus{
		EEBus:  eb,
		Helper: request.NewHelper(log),
		uri:    uri,
	}

	// REST API features require IP and credentials
	if ip != "" && user != "" && password != "" {
		ts, err := ghostone.TokenSource(ctx, log, wb.uri, user, password)
		if err != nil {
			return nil, err
		}

		wb.Client.Transport = &oauth2.Transport{
			Source: ts,
			Base:   transport.Insecure(),
		}

		// warn if PV optimization is active
		var pvMode ghostone.PvOptimizationMode
		if err := wb.getJSONCtx(ctx, wb.uri+"/charging/pvoptimization/mode", &pvMode); err == nil && pvMode.Value != ghostone.PvModeNone {
			log.WARN.Printf("wallbox PV optimization is active (%s), should be disabled when using evcc", pvMode.Value)
		}

		// warn if phase switching is disabled
		var relaisEnabled ghostone.Enabled
		if err := wb.getJSONCtx(ctx, wb.uri+"/system/relais-switch/enabled", &relaisEnabled); err == nil && !relaisEnabled.Enabled {
			log.WARN.Println("phase switching is disabled, enable it in the wallbox settings to use 1p3p switching")
		}

		// always wire up phase switching and RFID - the wallbox will reject
		// operations at runtime if the feature is disabled or not possible
		implement.Has(wb, implement.PhaseSwitcher(wb.phases1p3p))
		implement.Has(wb, implement.PhaseGetter(wb.getPhases))
		wb.hasRFID = true
	}

	// EEBus meter capabilities
	if hasMeter {
		implement.Has(wb, implement.Meter(eb.currentPower))
		implement.Has(wb, implement.PhaseCurrents(eb.currents))
		if hasChargedEnergy {
			implement.Has(wb, implement.ChargeRater(eb.chargedEnergy))
		}
	}

	return wb, nil
}

var _ api.Identifier = (*GhostEEBus)(nil)

// Identify implements api.Identifier, preferring RFID over EEBUS identification
func (wb *GhostEEBus) Identify() (string, error) {
	if wb.hasRFID {
		if id, err := wb.identify(); err == nil && id != "" {
			return id, nil
		}
	}
	return wb.EEBus.Identify()
}

// getJSONCtx executes a context-aware GET request and decodes the JSON response.
func (wb *GhostEEBus) getJSONCtx(ctx context.Context, url string, res any) error {
	req, err := request.New(http.MethodGet, url, nil, request.AcceptJSON)
	if err != nil {
		return err
	}
	return wb.DoJSON(req.WithContext(ctx), res)
}

// putJSON sends a PUT request with JSON body to the REST API.
func (wb *GhostEEBus) putJSON(url string, data any) error {
	req, err := request.New(http.MethodPut, url, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		_, err = wb.DoBody(req)
	}
	return err
}

// phases1p3p implements phase switching via REST API.
// Returns api.ErrNotAvailable when the EV communicates via ISO 15118,
// as relay switching would violate the high-level power contract.
func (wb *GhostEEBus) phases1p3p(phases int) error {
	evEntity, connected := wb.EEBus.isEvConnected()
	if connected {
		comStandard, err := wb.EEBus.cem.EvCC.CommunicationStandard(evEntity)
		if err != nil {
			return api.ErrNotAvailable
		}
		if comStandard != model.DeviceConfigurationKeyValueStringTypeIEC61851 {
			return api.ErrNotAvailable
		}
	}

	val := ghostone.RelaisStateOnePhase
	if phases == 3 {
		val = ghostone.RelaisStateThreePhase
	}

	if err := wb.putJSON(wb.uri+"/system/relais-switch/state", ghostone.RelaisSwitchStateWrite{Value: val}); err != nil {
		return err
	}

	// verify the switch was accepted by reading back the state
	var res ghostone.RelaisSwitchStateRead
	if err := wb.GetJSON(wb.uri+"/system/relais-switch/state", &res); err != nil {
		return err
	}

	switch res.Value {
	case ghostone.RelaisStateNotPossible, ghostone.RelaisStateNotAvailable:
		msg := fmt.Sprintf("phase switching denied: %s", res.Value)
		if res.LimitationReason != "" {
			msg += fmt.Sprintf(" (%s)", res.LimitationReason)
		}
		return errors.New(msg)
	}

	return nil
}

// getPhases implements phase reading via REST API.
func (wb *GhostEEBus) getPhases() (int, error) {
	var res ghostone.RelaisSwitchStateRead
	err := wb.GetJSON(wb.uri+"/system/relais-switch/state", &res)
	if err != nil {
		return 0, err
	}

	// check for transient or error states
	// "notPossible" is also returned if the EV is connected via ISO 15118, as phase switching is not allowed in that case
	switch res.Value {
	case ghostone.RelaisStateSwitchInProgress, ghostone.RelaisStateNotAvailable, ghostone.RelaisStateNotPossible:
		msg := fmt.Sprintf("phase switch state: %s", res.Value)
		if res.LimitationReason != "" {
			msg += fmt.Sprintf(" (%s)", res.LimitationReason)
		}
		// the phase state cannot be read - signal api.ErrNotAvailable so
		// callers treat it as a benign "not available" rather than an error
		wb.log.TRACE.Println(msg)
		return 0, api.ErrNotAvailable
	}

	switch res.CurrentState {
	case ghostone.RelaisStateOnePhase:
		return 1, nil
	case ghostone.RelaisStateThreePhase:
		return 3, nil
	default:
		return 0, fmt.Errorf("unknown phase state: %s", res.CurrentState)
	}
}

// identify implements RFID identification via REST API.
func (wb *GhostEEBus) identify() (string, error) {
	var res ghostone.RfidCardLastRead
	err := wb.GetJSON(wb.uri+"/rfid-cards/last-read", &res)
	return res.UUID, err
}
