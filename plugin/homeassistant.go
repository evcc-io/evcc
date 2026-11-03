package plugin

import (
	"context"
	"errors"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/homeassistant"
)

// HomeAssistant reads and writes Home Assistant entity states
type HomeAssistant struct {
	conn   *homeassistant.Connection
	entity string
}

func init() {
	registry.AddCtx("homeassistant", NewHomeAssistantPluginFromConfig)
}

// NewHomeAssistantPluginFromConfig creates a Home Assistant plugin
func NewHomeAssistantPluginFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	var cc struct {
		homeassistant.Config `mapstructure:",squash"`
		Entity               string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Entity == "" {
		return nil, errors.New("missing entity")
	}

	conn, err := homeassistant.NewConnection(util.NewLogger("homeassistant"), cc.URI, cc.Home_, cc.Insecure)
	if err != nil {
		return nil, err
	}

	return &HomeAssistant{
		conn:   conn,
		entity: cc.Entity,
	}, nil
}

var _ StringGetter = (*HomeAssistant)(nil)

// StringGetter returns the entity state
func (p *HomeAssistant) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		res, err := p.conn.GetState(p.entity)
		return res.State, err
	}, nil
}

var _ FloatGetter = (*HomeAssistant)(nil)

// FloatGetter returns the entity state as float
func (p *HomeAssistant) FloatGetter() (func() (float64, error), error) {
	return func() (float64, error) {
		return p.conn.GetFloatState(p.entity)
	}, nil
}

var _ IntGetter = (*HomeAssistant)(nil)

// IntGetter returns the entity state as int
func (p *HomeAssistant) IntGetter() (func() (int64, error), error) {
	return func() (int64, error) {
		return p.conn.GetIntState(p.entity)
	}, nil
}

var _ BoolGetter = (*HomeAssistant)(nil)

// BoolGetter returns the entity state as bool
func (p *HomeAssistant) BoolGetter() (func() (bool, error), error) {
	return func() (bool, error) {
		return p.conn.GetBoolState(p.entity)
	}, nil
}

var _ BoolSetter = (*HomeAssistant)(nil)

// BoolSetter invokes the entity's turn_on/turn_off service
func (p *HomeAssistant) BoolSetter(param string) (func(bool) error, error) {
	return func(enable bool) error {
		return p.conn.CallSwitchService(p.entity, enable)
	}, nil
}
