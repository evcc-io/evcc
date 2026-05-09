package iobroker

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// Connection represents a Iobroker API connection
type Connection struct {
	*request.Helper
	user     string
	password string
	instance *proxyInstance
}

var connections map[string]*Connection

func init() {
	connections = make(map[string]*Connection)
}

func GetConnection(name string) *Connection {
	return connections[name]
}

// NewConnection creates a new Iobroker connection
func NewConnection(log *util.Logger, name, uri, username, password string) error {
	if uri == "" {
		return errors.New("missing uri")
	}

	if username == "" || password == "" {
		return errors.New("invalid username or password")
	}

	if connections[name] != nil {
		return errors.New("duplicate name")
	}

	c := &Connection{
		Helper:   request.NewHelper(log),
		user:     username,
		password: password,
		instance: &proxyInstance{
			uri: util.DefaultScheme(util.DefaultPort(strings.TrimSuffix(uri, "/"), 8082), "http"),
		},
	}

	// Set up authentication headers
	c.Client.Transport = &oauth2.Transport{
		Base:   c.Client.Transport,
		Source: c.instance,
	}

	connections[name] = c

	return nil
}

// URI returns the base URI of the Home Assistant instance
func (c *Connection) URI() string {
	return c.instance.URI()
}

// GetState retrieves the state of an entity
func (c *Connection) GetState(entity string) (StateResponse, error) {
	var res StateResponse
	uri := fmt.Sprintf("%s/rest-api/v1/state/%s", c.instance.URI(), url.PathEscape(entity))

	if err := c.GetJSON(uri, &res); err != nil {
		return res, err
	}

	return res, nil
}

// GetIntState retrieves the state of an entity as int64
func (c *Connection) GetIntState(entity string) (int64, error) {
	state, err := c.GetState(entity)
	if err != nil {
		return 0, err
	}
	var value int64
	switch v := state.VAL.(type) {
	case string:
		value, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid numeric state '%s' for entity %s: %w", v, entity, err)
		}
	case float64:
		value = int64(math.Round(v))
	case float32:
		value = int64(math.Round(float64(v)))
	case int64:
		value = v
	default:
		return 0, fmt.Errorf("unknown type for entity %s: %w", entity, err)
	}
	return value, nil
}

// GetFloatState retrieves the state of an entity as float64
func (c *Connection) GetFloatState(entity string) (float64, error) {
	state, err := c.GetState(entity)
	if err != nil {
		return 0, err
	}
	var value float64
	switch v := state.VAL.(type) {
	case string:
		value, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid numeric state '%s' for entity %s: %w", v, entity, err)
		}
	case float64:
		value = v
	case float32:
		value = float64(v)
	case int64:
		value = float64(v)
	default:
		return 0, fmt.Errorf("unknown type for entity %s: %w", entity, err)
	}
	return value, nil
}

// GetBoolState retrieves the state of an entity as boolean
func (c *Connection) GetBoolState(entity string) (bool, error) {
	state, err := c.GetState(entity)
	if err != nil {
		return false, err
	}

	var value bool
	switch v := state.VAL.(type) {
	case string:
		res := strings.ToLower(v)
		switch res {
		case "on", "true", "1", "active", "yes":
			value = true
		case "off", "false", "0", "inactive", "no":
			value = false
		default:
			return false, fmt.Errorf("invalid boolean state '%s' for entity %s", state, entity)
		}
	case float64:
		value = math.Abs(v) > 0.01
	case int64:
		value = (v != 0)
	case bool:
		value = v
	default:
		return false, fmt.Errorf("unknown type for entity %s: %w", entity, err)
	}
	return value, nil
}

// GetStringState retrieves the state of an entity as time
func (c *Connection) GetStringState(entity string) (string, error) {
	state, err := c.GetState(entity)
	if err != nil {
		return "", err
	}

	var value string
	switch v := state.VAL.(type) {
	case string:
		value = v
	default:
		value = fmt.Sprintf("%v", v)
	}

	return value, nil
}

// GetTimeState retrieves the state of an entity as time
func (c *Connection) GetTimeState(entity string) (time.Time, error) {
	state, err := c.GetState(entity)
	if err != nil {
		return time.Time{}, err
	}

	switch v := state.VAL.(type) {
	case string:
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			return time.Unix(ts, 0), nil
		} else {
			return time.Parse(time.RFC3339, v)
		}
	case int64:
		return time.Unix(v, 0), nil
	case float64:
		return time.Unix(int64(v), 0), nil
	default:
		return time.Unix(0, 0), fmt.Errorf("unknown type for entity %s: %w", entity, err)
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
	"paused":             api.StatusB,

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
	value, err := c.GetStringState(entity)
	if err != nil {
		return api.StatusNone, err
	}

	if status, ok := chargeStatusMap[strings.ToLower(strings.TrimSpace(value))]; ok {
		return status, nil
	}

	return api.StatusNone, fmt.Errorf("unknown charge status: %s", value)
}

func domain(entity string) (string, error) {
	domain, _, ok := strings.Cut(entity, ".")
	if !ok {
		return "", fmt.Errorf("invalid entity format: %s", entity)
	}

	return domain, nil
}

func (c *Connection) SetState(entity string, value string) (SetStateResponse, error) {
	var res SetStateResponse
	state := fmt.Sprintf("{val=%s,ack=true}", value)
	uri := fmt.Sprintf("%s/rest-api/command/setState/%s&state=%s", c.instance.URI(), url.PathEscape(entity), url.PathEscape(state))

	if err := c.GetJSON(uri, &res); err != nil {
		return res, err
	}

	return res, nil
}

// CallSwitchService is a convenience method for switch services
func (c *Connection) SetBoolState(entity string, turnOn bool) error {
	var state string
	if turnOn {
		state = "true"
	} else {
		state = "false"
	}
	_, err := c.SetState(entity, state)
	return err
}

// CallNumberService is a convenience method for setting number entity values
func (c *Connection) SetFloatState(entity string, value float64) error {
	var sVal string
	sVal = fmt.Sprintf("%g", value)
	_, err := c.SetState(entity, sVal)
	return err
}
