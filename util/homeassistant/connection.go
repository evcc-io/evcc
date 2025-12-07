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
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

// Connection represents a Home Assistant API connection
type Connection struct {
	*request.Helper
	instance *proxyInstance
}

// NewConnection creates a new Home Assistant connection
func NewConnection(log *util.Logger, uri, home string) (*Connection, error) {
	if home != "" {
		log.WARN.Printf("using deprecated 'home' parameter '%s', please use 'uri' instead", home)
	}

	if uri == "" && home == "" {
		return nil, errors.New("missing either uri or home")
	}

	c := &Connection{
		Helper: request.NewHelper(log),
		instance: &proxyInstance{
			home: home,
			uri:  uri,
		},
	}

	// Set up authentication headers
	c.Client.Transport = &oauth2.Transport{
		Base:   c.Client.Transport,
		Source: c.instance,
	}

	return c, nil
}

// GetStates retrieves the list of entities
func (c *Connection) GetStates() ([]StateResponse, error) {
	var res []StateResponse
	uri := fmt.Sprintf("%s/api/states", c.instance.URI())

	err := c.GetJSON(uri, &res)

	return res, err
}

// GetState retrieves the state of an entity
func (c *Connection) GetState(entity string) (string, error) {
	var res StateResponse
	uri := fmt.Sprintf("%s/api/states/%s", c.instance.URI(), url.PathEscape(entity))

	if err := c.GetJSON(uri, &res); err != nil {
		return "", err
	}

	if res.State == "unknown" || res.State == "unavailable" {
		return "", api.ErrNotAvailable
	}

	return res.State, nil
}

// GetIntState retrieves the state of an entity as int64
func (c *Connection) GetIntState(entity string) (int64, error) {
	state, err := c.GetState(entity)
	if err != nil {
		return 0, err
	}

	value, err := strconv.ParseInt(state, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric state '%s' for entity %s: %w", state, entity, err)
	}

	return value, nil
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
	"no_power":           api.StatusB,
	"complete":           api.StatusB,
	"stopped":            api.StatusB,
	"starting":           api.StatusB,

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

// GetChargeStatus maps Home Assistant states to api.ChargeStatus
func (c *Connection) GetChargeStatus(entity string) (api.ChargeStatus, error) {
	state, err := c.GetState(entity)
	if err != nil {
		return api.StatusNone, err
	}

	if status, ok := chargeStatusMap[strings.ToLower(strings.TrimSpace(state))]; ok {
		return status, nil
	}

	return api.StatusNone, fmt.Errorf("unknown charge status: %s", state)
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

	data := map[string]any{
		"entity_id": entity,
	}

	return c.CallService(domain, service, data)
}

// CallNumberService is a convenience method for setting number entity values
func (c *Connection) CallNumberService(entity string, value float64) error {
	data := map[string]any{
		"entity_id": entity,
		"value":     value,
	}

	return c.CallService("number", "set_value", data)
}

// GetPhaseFloatStates retrieves three phase values (currents, voltages, etc.)
func (c *Connection) GetPhaseFloatStates(entities []string) (float64, float64, float64, error) {
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
		return nil, fmt.Errorf("must contain three-phase entities (L1, L2, L3), got %d", len(entities))
	}
}
