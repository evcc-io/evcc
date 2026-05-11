package iobroker

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

// Connection represents a Iobroker API connection
type Connection struct {
	*request.Helper
	Log      *util.Logger
	Identity *Identity
}

var connections map[string]*Connection

func init() {
	connections = make(map[string]*Connection)
}

func GetConnection(name string) *Connection {
	return connections[name]
}

func NewConnection(log *util.Logger, name, uri, username, password string) error {
	if connections[name] != nil {
		return errors.New("duplicate name")
	}

	identity, err := NewIdentity(
		log,
		util.DefaultScheme(strings.TrimSuffix(uri, "/"), "http"),
		username,
		password)
	if err != nil {
		return err
	}
	c := &Connection{
		Helper:   request.NewHelper(log),
		Log:      log,
		Identity: identity,
	}

	// Set up authentication headers
	c.Client.Transport = &oauth2.Transport{
		Base:   c.Client.Transport,
		Source: c.Identity,
	}

	connections[name] = c
	log.DEBUG.Println("Created connection " + name)

	return nil
}

// URI returns the base URI of the iobroker instance
func (c *Connection) URI() string {
	return c.Identity.uri
}

// GetState retrieves the state of an entity
func (c *Connection) GetState(entity string) (StateResponse, error) {
	var res StateResponse
	uri := fmt.Sprintf("%s/rest-api/v1/state/%s", c.URI(), url.PathEscape(entity))

	err := c.GetJSON(uri, &res)
	return res, err
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
			return 0, fmt.Errorf("invalid numeric state '%s' for entity %s", v, entity)
		}
	case float64:
		value = int64(math.Round(v))
	case float32:
		value = int64(math.Round(float64(v)))
	case int64:
		value = v
	default:
		return 0, fmt.Errorf("unknown type for entity %s", entity)
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
		return 0, fmt.Errorf("unknown type for entity %s", entity)
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
			return false, fmt.Errorf("invalid boolean state '%s' for entity %s", v, entity)
		}
	case float64:
		value = math.Abs(v) > 0.01
	case int64:
		value = (v != 0)
	case bool:
		value = v
	default:
		return false, fmt.Errorf("unknown type for entity %s", entity)
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
		return time.Unix(0, 0), fmt.Errorf("unknown type for entity %s", entity)
	}
}

// chargeStatusMap maps Iobroker states to EVCC charge status
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

// GetChargeStatus maps Iobroker states to api.ChargeStatus
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

func (c *Connection) SetState(entity string, value any) (SetStateResponse, error) {
	var res SetStateResponse

	state, err := json.Marshal(SetValueRequest{Value: value, Ack: false})
	uri := fmt.Sprintf("%s/rest-api/v1/command/setState", c.URI())

	if err != nil {
		return res, err
	}

	params := url.Values{}
	params.Add("id", entity)
	params.Add("state", string(state))

	req, err := request.New(http.MethodPost, uri, strings.NewReader(params.Encode()), map[string]string{
		"Content-Type": request.FormContent,
		"Accept":       "application/json, text/plain",
	})

	if err == nil {
		err = c.DoJSON(req, &res)
	}

	return res, err
}

// SetBoolState is a convenience method for boolean objects
func (c *Connection) SetBoolState(entity string, turnOn bool) error {
	_, err := c.SetState(entity, turnOn)
	return err
}

// SetFloatState is a convenience method for setting number entity values
func (c *Connection) SetFloatState(entity string, value float64) error {
	_, err := c.SetState(entity, value)
	return err
}

// SetIntState is a convenience method for setting number entity values
func (c *Connection) SetIntState(entity string, value int64) error {
	_, err := c.SetState(entity, value)
	return err
}

// SetStringState is a convenience method for setting string entity values
func (c *Connection) SetStringState(entity string, value string) error {
	_, err := c.SetState(entity, value)
	return err
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
