package charger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	enodeProductionAPI   = "https://enode-api.production.enode.io"
	enodeProductionToken = "https://oauth.production.enode.io/oauth2/token"
	enodeSandboxAPI      = "https://enode-api.sandbox.enode.io"
	enodeSandboxToken    = "https://oauth.sandbox.enode.io/oauth2/token"

	enodeDefaultCache         = time.Second
	enodeDefaultActionTimeout = 45 * time.Second
	enodeActionPollInterval   = time.Second
)

type enodeEnvironment struct {
	apiURL   string
	tokenURL string
}

type enodeProblem struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

func (p *enodeProblem) Error() string {
	return p.Detail
}

type enodePagination struct {
	After  *string `json:"after"`
	Before *string `json:"before"`
}

type enodeCapability struct {
	IsCapable bool `json:"isCapable"`
}

type enodeChargerCapabilities struct {
	StartCharging enodeCapability `json:"startCharging"`
	StopCharging  enodeCapability `json:"stopCharging"`
	SetMaxCurrent enodeCapability `json:"setMaxCurrent"`
}

type enodeChargerInformation struct {
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	SerialNumber string `json:"serialNumber"`
}

type enodeChargerChargeState struct {
	IsPluggedIn        *bool      `json:"isPluggedIn"`
	IsCharging         *bool      `json:"isCharging"`
	ChargeRate         *float64   `json:"chargeRate"`
	LastUpdated        *time.Time `json:"lastUpdated"`
	MaxCurrent         *float64   `json:"maxCurrent"`
	PowerDeliveryState string     `json:"powerDeliveryState"`
}

type enodeCharger struct {
	ID           string                   `json:"id"`
	UserID       string                   `json:"userId"`
	Vendor       string                   `json:"vendor"`
	IsReachable  bool                     `json:"isReachable"`
	ChargeState  enodeChargerChargeState  `json:"chargeState"`
	Capabilities enodeChargerCapabilities `json:"capabilities"`
	Information  enodeChargerInformation  `json:"information"`
}

type enodeChargerList struct {
	Data       []enodeCharger  `json:"data"`
	Pagination enodePagination `json:"pagination"`
}

type enodeActionFailureReason struct {
	Type   string `json:"type"`
	Detail string `json:"detail"`
}

type enodeAction struct {
	ID            string                    `json:"id"`
	State         string                    `json:"state"`
	FailureReason *enodeActionFailureReason `json:"failureReason"`
	Kind          string                    `json:"kind"`
}

// Enode controls a charger via the Enode cloud API.
type Enode struct {
	*request.Helper
	baseURL              string
	charger              string
	enabled              bool
	configuredMaxCurrent *float64
	statusG              util.Cacheable[enodeCharger]
	timeout              time.Duration
	actionTO             time.Duration
}

func init() {
	registry.AddCtx("enode", NewEnodeFromConfig)
}

func NewEnodeFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		ClientID     string
		ClientSecret string
		UserID       string
		ChargerID    string
		APIURL       string
		Timeout      time.Duration
		Cache        time.Duration
	}{
		Timeout: request.Timeout,
		Cache:   enodeDefaultCache,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.ClientID == "" || cc.ClientSecret == "" {
		return nil, api.ErrMissingCredentials
	}

	env := resolveEnodeEnvironment(cc.APIURL)

	return newEnode(ctx, env, cc.ClientID, cc.ClientSecret, cc.UserID, cc.ChargerID, cc.Cache, cc.Timeout)
}

func resolveEnodeEnvironment(apiURL string) enodeEnvironment {
	env := enodeEnvironment{apiURL: enodeProductionAPI, tokenURL: enodeProductionToken}

	if apiURL != "" {
		normalizedAPIURL := strings.TrimRight(strings.TrimSpace(apiURL), "/")
		env.apiURL = normalizedAPIURL

		if inferredTokenURL, ok := inferEnodeTokenURLFromAPI(normalizedAPIURL); ok {
			env.tokenURL = inferredTokenURL
		}
	}

	return env
}

func inferEnodeTokenURLFromAPI(apiURL string) (string, bool) {
	parsed, err := url.Parse(apiURL)
	if err != nil {
		return "", false
	}

	host := strings.ToLower(parsed.Hostname())
	switch host {
	case "enode-api.sandbox.enode.io":
		return enodeSandboxToken, true
	case "enode-api.production.enode.io":
		return enodeProductionToken, true
	default:
		return "", false
	}
}

func newEnode(ctx context.Context, env enodeEnvironment, clientID, clientSecret, userID, chargerID string, cache, timeout time.Duration) (*Enode, error) {
	log := util.NewLogger("enode").Redact(clientID, clientSecret)

	oauthConfig := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     env.tokenURL,
	}

	client := request.NewHelper(log)
	client.Client = oauthConfig.Client(ctx)
	client.Client.Timeout = timeout

	wb := &Enode{
		Helper:   client,
		baseURL:  env.apiURL,
		timeout:  timeout,
		actionTO: enodeDefaultActionTimeout,
		enabled:  true,
	}

	charger, err := wb.resolveCharger(userID, chargerID)
	if err != nil {
		return nil, err
	}

	wb.charger = charger.ID
	if s := interpretEnodeState(charger.ChargeState); s.known {
		wb.enabled = s.enabled
	}
	wb.updateMaxCurrent(charger.ChargeState.MaxCurrent)
	wb.statusG = util.ResettableCached(func() (enodeCharger, error) {
		var res enodeCharger
		if err := wb.getJSON("/chargers/"+wb.charger, &res); err != nil {
			return res, err
		}
		if s := interpretEnodeState(res.ChargeState); s.known {
			wb.enabled = s.enabled
		}
		wb.updateMaxCurrent(res.ChargeState.MaxCurrent)
		return res, nil
	}, cache)

	return wb, nil
}

func (wb *Enode) resolveCharger(userID, chargerID string) (enodeCharger, error) {
	chargers, err := wb.listChargers(userID)
	if err != nil {
		return enodeCharger{}, err
	}

	if len(chargers) == 0 {
		return enodeCharger{}, backoff.Permanent(errors.New("no linked chargers found in Enode account"))
	}

	if chargerID != "" {
		for _, charger := range chargers {
			if strings.EqualFold(charger.ID, chargerID) {
				return charger, nil
			}
		}
		return enodeCharger{}, fmt.Errorf("cannot find charger, got: %v", chargerIDs(chargers))
	}

	if len(chargers) == 1 {
		return chargers[0], nil
	}

	return enodeCharger{}, fmt.Errorf("multiple chargers found, set chargerid explicitly: %v", chargerIDs(chargers))
}

func chargerIDs(chargers []enodeCharger) []string {
	res := make([]string, 0, len(chargers))
	for _, charger := range chargers {
		res = append(res, charger.ID)
	}
	return res
}

func (wb *Enode) listChargers(userID string) ([]enodeCharger, error) {
	path := "/chargers"
	if userID != "" {
		path = "/users/" + userID + "/chargers"
	}

	var res enodeChargerList
	if err := wb.getJSON(path, &res); err != nil {
		return nil, err
	}

	return res.Data, nil
}

func (wb *Enode) getJSON(path string, out any) error {
	return wb.doJSON(http.MethodGet, path, nil, out)
}

func (wb *Enode) postJSON(path string, body any, out any) error {
	return wb.doJSON(http.MethodPost, path, body, out)
}

func (wb *Enode) doJSON(method, path string, body any, out any) error {
	var payload io.Reader
	if body != nil {
		payload = request.MarshalJSON(body)
	}

	headers := request.AcceptJSON
	if body != nil {
		headers = request.JSONEncoding
	}

	req, err := request.New(method, wb.baseURL+path, payload, headers)
	if err != nil {
		return err
	}

	resp, err := wb.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var problem enodeProblem
		_ = json.NewDecoder(resp.Body).Decode(&problem)

		if problem.Detail != "" {
			return backoff.Permanent(&problem)
		}

		return backoff.Permanent(request.NewStatusError(resp))
	}

	if resp.StatusCode == http.StatusNoContent || out == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (wb *Enode) Status() (api.ChargeStatus, error) {
	status, err := wb.statusG.Get()
	if err != nil {
		return api.StatusNone, err
	}

	s := interpretEnodeState(status.ChargeState)
	return s.status, nil
}

func (wb *Enode) Enabled() (bool, error) {
	status, err := wb.statusG.Get()
	if err != nil {
		return false, err
	}

	s := interpretEnodeState(status.ChargeState)
	if !s.known {
		return false, api.ErrMustRetry
	}

	wb.enabled = s.enabled

	return verifyEnabled(wb, s.enabled)
}

func (wb *Enode) Enable(enable bool) error {
	status, err := wb.statusG.Get()
	if err != nil {
		return err
	}

	if s := interpretEnodeState(status.ChargeState); s.known && enable == s.enabled {
		return nil
	}

	capable := status.Capabilities.StopCharging.IsCapable
	action := "STOP"
	if enable {
		capable = status.Capabilities.StartCharging.IsCapable
		action = "START"
	}

	if !capable {
		return api.ErrNotAvailable
	}

	var res enodeAction
	if err := wb.postJSON("/chargers/"+wb.charger+"/charging", struct {
		Action string `json:"action"`
	}{Action: action}, &res); err != nil {
		if isEnodeNoOpError(err, enable) {
			wb.enabled = enable
			wb.statusG.Reset()
			return nil
		}
		return err
	}

	if err := wb.awaitAction(res.ID); err != nil {
		return err
	}

	wb.enabled = enable
	wb.statusG.Reset()
	return nil
}

func (wb *Enode) MaxCurrent(current int64) error {
	status, err := wb.statusG.Get()
	if err != nil {
		return err
	}

	if !status.Capabilities.SetMaxCurrent.IsCapable {
		return api.ErrNotAvailable
	}

	if status.ChargeState.MaxCurrent != nil && int64(*status.ChargeState.MaxCurrent) == current {
		return nil
	}

	var res enodeAction
	if err := wb.postJSON("/chargers/"+wb.charger+"/max-current", struct {
		MaxCurrent float64 `json:"maxCurrent"`
	}{MaxCurrent: float64(current)}, &res); err != nil {
		if isEnodeAlreadySetCurrent(err) {
			wb.updateMaxCurrent(ptr(float64(current)))
			wb.statusG.Reset()
			return nil
		}
		return err
	}

	if err := wb.awaitAction(res.ID); err != nil {
		return err
	}

	wb.updateMaxCurrent(ptr(float64(current)))
	wb.statusG.Reset()
	return nil
}

func (wb *Enode) awaitAction(actionID string) error {
	deadline := time.Now().Add(wb.actionTO)

	for time.Now().Before(deadline) {
		var res enodeAction
		if err := wb.getJSON("/chargers/actions/"+actionID, &res); err != nil {
			return err
		}

		switch res.State {
		case "CONFIRMED":
			return nil
		case "FAILED", "CANCELLED":
			if res.FailureReason != nil && res.FailureReason.Detail != "" {
				return backoff.Permanent(fmt.Errorf("%s", res.FailureReason.Detail))
			}
			return backoff.Permanent(fmt.Errorf("enode action %s", strings.ToLower(res.State)))
		}

		time.Sleep(enodeActionPollInterval)
	}

	return api.ErrTimeout
}

func (wb *Enode) updateMaxCurrent(maxCurrent *float64) {
	if maxCurrent != nil {
		wb.configuredMaxCurrent = maxCurrent
	}
}

type enodeState struct {
	status  api.ChargeStatus
	enabled bool
	known   bool
}

func interpretEnodeState(cs enodeChargerChargeState) enodeState {
	if cs.IsCharging != nil && *cs.IsCharging {
		return enodeState{status: api.StatusC, enabled: true, known: true}
	}

	switch cs.PowerDeliveryState {
	case "UNPLUGGED":
		return enodeState{status: api.StatusA, enabled: false, known: true}
	case "PLUGGED_IN:INITIALIZING", "PLUGGED_IN:STOPPED", "PLUGGED_IN:NO_POWER", "PLUGGED_IN:FAULT":
		return enodeState{status: api.StatusB, enabled: cs.PowerDeliveryState != "PLUGGED_IN:STOPPED", known: true}
	case "PLUGGED_IN:CHARGING", "PLUGGED_IN:DISCHARGING":
		return enodeState{status: api.StatusC, enabled: true, known: true}
	default:
		if cs.IsPluggedIn != nil {
			if *cs.IsPluggedIn {
				return enodeState{status: api.StatusB, enabled: true, known: true}
			}
			return enodeState{status: api.StatusA, enabled: false, known: true}
		}
		return enodeState{status: api.StatusNone, enabled: false, known: false}
	}
}

var _ api.Meter = (*Enode)(nil)

func (wb *Enode) CurrentPower() (float64, error) {
	status, err := wb.statusG.Get()
	if err != nil {
		return 0, err
	}

	if status.ChargeState.ChargeRate == nil {
		return 0, nil
	}

	return *status.ChargeState.ChargeRate * 1e3, nil
}

var _ api.CurrentGetter = (*Enode)(nil)

func (wb *Enode) GetMaxCurrent() (float64, error) {
	status, err := wb.statusG.Get()
	if err != nil {
		return 0, err
	}

	if status.ChargeState.MaxCurrent == nil {
		if wb.configuredMaxCurrent != nil {
			return *wb.configuredMaxCurrent, nil
		}
		return 0, api.ErrMustRetry
	}

	return *status.ChargeState.MaxCurrent, nil
}

func ptr[T any](v T) *T {
	return &v
}

func isEnodeNoOpError(err error, enable bool) bool {
	var problem *enodeProblem
	if !errors.As(err, &problem) {
		return false
	}

	detail := strings.ToLower(problem.Detail)
	if enable {
		return strings.Contains(detail, "already charging") || strings.Contains(detail, "already started")
	}

	return strings.Contains(detail, "already stopped")
}

func isEnodeAlreadySetCurrent(err error) bool {
	var problem *enodeProblem
	if !errors.As(err, &problem) {
		return false
	}

	return strings.Contains(strings.ToLower(problem.Detail), "already at the current setting")
}
