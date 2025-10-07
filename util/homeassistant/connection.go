package homeassistant

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// Connection represents a Home Assistant API connection
type Connection struct {
	*request.Helper
	uri string
}

// NewConnection creates a new Home Assistant connection
func NewConnection(baseURL, token string) (*Connection, error) {
	if baseURL == "" {
		return nil, errors.New("missing baseURL")
	}
	if token == "" {
		return nil, errors.New("missing token")
	}

	log := util.NewLogger("homeassistant")
	c := &Connection{
		Helper: request.NewHelper(log.Redact(token)),
		uri:    strings.TrimSuffix(baseURL, "/"),
	}

	// Set up authentication headers
	c.Client.Transport = &transport.Decorator{
		Base: c.Client.Transport,
		Decorator: transport.DecorateHeaders(map[string]string{
			"Authorization": "Bearer " + token,
			"Content-Type":  "application/json",
		}),
	}

	return c, nil
}

// StateResponse represents a Home Assistant entity state
type StateResponse struct {
	State      string                 `json:"state"`
	Attributes map[string]interface{} `json:"attributes"`
}

// GetState retrieves the state of an entity
func (c *Connection) GetState(entity string) (string, error) {
	var res StateResponse
	uri := fmt.Sprintf("%s/api/states/%s", c.uri, url.PathEscape(entity))

	if err := c.GetJSON(uri, &res); err != nil {
		return "", err
	}

	if res.State == "unknown" || res.State == "unavailable" {
		return "", api.ErrNotAvailable
	}

	return res.State, nil
}

// GetFloatState retrieves the state of an entity as float64
func (c *Connection) GetFloatState(entity string) (float64, error) {
	state, err := c.GetState(entity)
	if err != nil {
		return 0, err
	}

	value, err := strconv.ParseFloat(state, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric state '%s' for entity %s: %w", state, entity, err)
	}

	return value, nil
}

// GetBoolState retrieves the state of an entity as boolean
func (c *Connection) GetBoolState(entity string) (bool, error) {
	state, err := c.GetState(entity)
	if err != nil {
		return false, err
	}

	state = strings.ToLower(state)
	switch state {
	case "on", "true", "1", "active", "yes":
		return true, nil
	case "off", "false", "0", "inactive", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean state '%s' for entity %s", state, entity)
	}
}

// CallService calls a Home Assistant service
func (c *Connection) CallService(domain, service string, data map[string]interface{}) error {
	uri := fmt.Sprintf("%s/api/services/%s/%s", c.uri, domain, service)

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	_, err = c.DoBody(req)
	return err
}

// CallSwitchService is a convenience method for switch services
func (c *Connection) CallSwitchService(entity string, turnOn bool) error {
	parts := strings.Split(entity, ".")
	if len(parts) == 0 {
		return fmt.Errorf("invalid entity format: %s", entity)
	}

	domain := parts[0]
	service := "turn_off"
	if turnOn {
		service = "turn_on"
	}

	data := map[string]interface{}{
		"entity_id": entity,
	}

	return c.CallService(domain, service, data)
}

// CallNumberService is a convenience method for setting number entity values
func (c *Connection) CallNumberService(entity string, value float64) error {
	data := map[string]interface{}{
		"entity_id": entity,
		"value":     value,
	}

	return c.CallService("number", "set_value", data)
}

// GetPhaseStates retrieves three phase values (currents, voltages, etc.)
func (c *Connection) GetPhaseStates(entities []string) (float64, float64, float64, error) {
	if len(entities) != 3 {
		return 0, 0, 0, errors.New("invalid phase entities")
	}

	var l1, l2, l3 float64
	var err error

	if l1, err = c.GetFloatState(entities[0]); err != nil {
		return 0, 0, 0, fmt.Errorf("phase L1: %w", err)
	}

	if l2, err = c.GetFloatState(entities[1]); err != nil {
		return 0, 0, 0, fmt.Errorf("phase L2: %w", err)
	}

	if l3, err = c.GetFloatState(entities[2]); err != nil {
		return 0, 0, 0, fmt.Errorf("phase L3: %w", err)
	}

	return l1, l2, l3, nil
}

// ValidatePhaseEntities validates that phase entity arrays contain 1 or 3 entities
func ValidatePhaseEntities(entities []string) ([]string, error) {
	switch len(entities) {
	case 0:
		return nil, nil
	case 3:
		return entities, nil
	default:
		return nil, fmt.Errorf("must contain three-phase entities (L1, L2, L3), got %d", len(entities))
	}
}

// chargeStatusMap maps Home Assistant states to EVCC charge status
var chargeStatusMap = map[string]api.ChargeStatus{
	// Status C - Charging
	"c":        api.StatusC,
	"charging": api.StatusC,
	"on":       api.StatusC,
	"true":     api.StatusC,
	"active":   api.StatusC,
	"1":        api.StatusC,

	// Status B - Connected/Ready
	"b":                  api.StatusB,
	"connected":          api.StatusB,
	"ready":              api.StatusB,
	"plugged":            api.StatusB,
	"charging_completed": api.StatusB,
	"initialising":       api.StatusB,
	"preparing":          api.StatusB,
	"2":                  api.StatusB,

	// Status A - Disconnected
	"a":                   api.StatusA,
	"disconnected":        api.StatusA,
	"off":                 api.StatusA,
	"none":                api.StatusA,
	"unavailable":         api.StatusA,
	"unknown":             api.StatusA,
	"notreadyforcharging": api.StatusA,
	"not_plugged":         api.StatusA,
	"0":                   api.StatusA,
}

// ParseChargeStatus maps Home Assistant states to EVCC charge status
func ParseChargeStatus(state string) (api.ChargeStatus, error) {
	normalized := strings.ToLower(strings.TrimSpace(state))
	if status, ok := chargeStatusMap[normalized]; ok {
		return status, nil
	}

	return api.StatusNone, fmt.Errorf("unknown charge status: %s", state)
}
