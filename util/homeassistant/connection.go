package homeassistant

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

// Connection represents a Home Assistant API connection
type Connection struct {
	*request.Helper
	instance *proxyInstance
}

// NewConnection creates a new Home Assistant connection
func NewConnection(log *util.Logger, uri, home string, insecure bool) (*Connection, error) {
	if home != "" {
		log.WARN.Printf("using deprecated 'home' parameter '%s', please use 'uri' instead", home)
	}

	if uri == "" && home == "" {
		return nil, errors.New("missing either uri or home")
	}

	c := &Connection{
		Helper: request.NewHelper(log),
		instance: &proxyInstance{
			home:     home,
			uri:      util.DefaultScheme(strings.TrimSuffix(uri, "/"), "http"),
			insecure: insecure,
		},
	}

	// override the transport to accept self-signed certificates
	if insecure {
		c.Client.Transport = request.NewTripper(log, transport.Insecure())
	}

	// Set up authentication headers
	c.Client.Transport = &oauth2.Transport{
		Base:   c.Client.Transport,
		Source: c.instance,
	}

	return c, nil
}

// URI returns the base URI of the Home Assistant instance
func (c *Connection) URI() string {
	return c.instance.URI()
}

// GetStates retrieves the list of entities
func (c *Connection) GetStates() ([]StateResponse, error) {
	var res []StateResponse
	uri := fmt.Sprintf("%s/api/states", c.instance.URI())
	err := c.GetJSON(uri, &res)
	return res, err
}

// GetServices retrieves the list of callable services
func (c *Connection) GetServices() ([]ServiceDomainResponse, error) {
	var res []ServiceDomainResponse
	uri := fmt.Sprintf("%s/api/services", c.instance.URI())
	err := c.GetJSON(uri, &res)
	return res, err
}

// GetState retrieves the state of an entity
func (c *Connection) GetState(entity string) (StateResponse, error) {
	var res StateResponse
	uri := fmt.Sprintf("%s/api/states/%s", c.instance.URI(), url.PathEscape(entity))

	if err := c.GetJSON(uri, &res); err != nil {
		return res, err
	}

	if res.State == "unknown" || res.State == "unavailable" {
		return res, api.ErrNotAvailable
	}

	return res, nil
}

// GetIntState retrieves the state of an entity as int64
func (c *Connection) GetIntState(entity string) (int64, error) {
	state, err := c.GetState(entity)
	if err != nil {
		return 0, err
	}

	value, err := strconv.ParseInt(state.State, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric state '%s' for entity %s: %w", state.State, entity, err)
	}

	return value, nil
}

// GetFloatState retrieves the state of an entity as float64
func (c *Connection) GetFloatState(entity string) (float64, error) {
	// leading minus sign?
	entity, invert := strings.CutPrefix(entity, "-")

	state, err := c.GetState(entity)
	if err != nil {
		return 0, err
	}

	value, err := strconv.ParseFloat(state.State, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric state '%s' for entity %s: %w", state.State, entity, err)
	}

	scale, err := state.scale()
	if err != nil {
		return 0, fmt.Errorf("%w for entity %s", err, entity)
	}

	if invert {
		value = -value
	}

	return scale * value, nil
}

// GetBoolState retrieves the state of an entity as boolean
func (c *Connection) GetBoolState(entity string) (bool, error) {
	state, err := c.GetState(entity)
	if err != nil {
		return false, err
	}

	res := strings.ToLower(state.State)
	switch res {
	case "on", "true", "1", "active", "yes":
		return true, nil
	case "off", "false", "0", "inactive", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean state '%s' for entity %s", state, entity)
	}
}

// GetTimeState retrieves the state of an entity as time
func (c *Connection) GetTimeState(entity string) (time.Time, error) {
	state, err := c.GetState(entity)
	if err != nil {
		return time.Time{}, err
	}

	if ts, err := strconv.ParseInt(state.State, 10, 64); err == nil {
		return time.Unix(ts, 0), nil
	}

	return time.Parse(time.RFC3339, state.State)
}

// chargeStatusMap maps unambiguous Home Assistant states to evcc charge status.
// Vendor-specific states are configured per device, see NewStatusMap.
var chargeStatusMap = map[string]api.ChargeStatus{
	"a":                  api.StatusA,
	"disconnected":       api.StatusA,
	"not_plugged":        api.StatusA,
	"b":                  api.StatusB,
	"connected":          api.StatusB,
	"plugged":            api.StatusB,
	"starting":           api.StatusB,
	"stopped":            api.StatusB,
	"paused":             api.StatusB,
	"complete":           api.StatusB,
	"charging_completed": api.StatusB,
	"c":                  api.StatusC,
	"charging":           api.StatusC,
}

// StatusMap maps device-specific Home Assistant states to evcc charge status
type StatusMap map[string]api.ChargeStatus

// NewStatusMap creates a status map from comma-separated, case-insensitive lists
// of states. It extends the built-in mapping, overriding it only for states
// explicitly mapped to a different status.
func NewStatusMap(a, b, c string) (StatusMap, error) {
	res := make(StatusMap)

	for _, e := range []struct {
		status api.ChargeStatus
		states string
	}{
		{api.StatusA, a},
		{api.StatusB, b},
		{api.StatusC, c},
	} {
		for s := range strings.SplitSeq(e.states, ",") {
			if s = strings.ToLower(strings.TrimSpace(s)); s != "" {
				if status, ok := res[s]; ok {
					return nil, fmt.Errorf("status %s: duplicate state '%s', already mapped to %s", e.status, s, status)
				}
				res[s] = e.status
			}
		}
	}

	return res, nil
}

// GetChargeStatus maps Home Assistant states to api.ChargeStatus. The
// device-specific status map extends the built-in mapping and takes precedence.
func (c *Connection) GetChargeStatus(entity string, states StatusMap) (api.ChargeStatus, error) {
	state, err := c.GetState(entity)
	if err != nil {
		return api.StatusNone, err
	}

	s := strings.ToLower(strings.TrimSpace(state.State))

	if status, ok := states[s]; ok {
		return status, nil
	}
	if status, ok := chargeStatusMap[s]; ok {
		return status, nil
	}

	return api.StatusNone, fmt.Errorf("unknown charge status '%s' for entity %s", state.State, entity)
}

// CallService calls a Home Assistant service
func (c *Connection) CallService(domain, service string, data map[string]any) error {
	uri := fmt.Sprintf("%s/api/services/%s/%s", c.instance.URI(), domain, service)

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	_, err = c.DoBody(req)
	return err
}

func domain(entity string) (string, error) {
	domain, _, ok := strings.Cut(entity, ".")
	if !ok {
		return "", fmt.Errorf("invalid entity format: %s", entity)
	}

	return domain, nil
}

// CallSwitchService is a convenience method for switch-like services. The
// service name depends on the entity domain: stateless button domains expose
// only `press`, while switch-style domains use `turn_on` / `turn_off`.
func (c *Connection) CallSwitchService(entity string, turnOn bool) error {
	domain, err := domain(entity)
	if err != nil {
		return err
	}

	var service string
	switch domain {
	case "button", "input_button":
		// Buttons are stateless — they only have a press action.
		if !turnOn {
			return fmt.Errorf("entity %s has no off action", entity)
		}
		service = "press"
	default:
		service = "turn_off"
		if turnOn {
			service = "turn_on"
		}
	}

	data := map[string]any{
		"entity_id": entity,
	}

	return c.CallService(domain, service, data)
}

// CallNumberService is a convenience method for setting number entity values
func (c *Connection) CallNumberService(entity string, value float64) error {
	domain, err := domain(entity)
	if err != nil {
		return err
	}

	data := map[string]any{
		"entity_id": entity,
		"value":     value,
	}

	return c.CallService(domain, "set_value", data)
}

// CallSelectService is a convenience method for setting select entity options.
func (c *Connection) CallSelectService(entity, option string) error {
	domain, err := domain(entity)
	if err != nil {
		return err
	}

	data := map[string]any{
		"entity_id": entity,
		"option":    option,
	}

	return c.CallService(domain, "select_option", data)
}

// GetPhaseFloatStates retrieves three phase values (currents, voltages, etc.)
func (c *Connection) GetPhaseFloatStates(entities []string) (float64, float64, float64, error) {
	if len(entities) != 3 {
		return 0, 0, 0, errors.New("invalid phase entities")
	}

	var res [3]float64

	for i := range res {
		f, err := c.GetFloatState(entities[i])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("phase L%d: %w", i+1, err)
		}
		res[i] = f
	}

	return res[0], res[1], res[2], nil
}

// ValidatePhaseEntities validates that phase entity arrays contain 1 or 3 entities
func ValidatePhaseEntities(phases []string) ([]string, error) {
	entities := lo.FilterMap(phases, func(s string, _ int) (string, bool) {
		t := strings.TrimSpace(s)
		return t, t != ""
	})

	switch len(entities) {
	case 0:
		return nil, nil
	case 3:
		return entities, nil
	default:
		return nil, errors.New("invalid phase entities")
	}
}
