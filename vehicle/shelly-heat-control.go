package vehicle

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/shelly"
	"github.com/evcc-io/evcc/util"
)

type Gen2Temperature struct {
	TC, TF        float64
	Code, Message string
}

type Shelly struct {
	*embed
	conn     *shelly.Connection
	targetTC float64
}

func init() {
	registry.Add("shelly-heat-control", NewShellyFromConfig)
}

func NewShellyFromConfig(other map[string]any) (api.Vehicle, error) {
	var vc struct {
		embed               `mapstructure:",squash"`
		URI, User, Password string
		Channel             int
		TargetTC            float64
	}

	if err := util.DecodeOther(other, &vc); err != nil {
		return nil, err
	}

	if vc.TargetTC == 0 {
		return nil, errors.New("target temperature cannot be 0")
	}
	return NewShelly(vc.embed, vc.URI, vc.User, vc.Password, vc.Channel, vc.TargetTC)
}

func NewShelly(embed embed, uri, user, password string, channel int, targettc float64) (*Shelly, error) {
	conn, err := shelly.NewConnection(uri, user, password, channel, time.Second)
	if err != nil {
		return nil, err
	}

	v := &Shelly{
		conn:     conn,
		embed:    &embed,
		targetTC: targettc,
	}

	var res shelly.Gen2Methods
	if err := v.conn.ExecCmd("Shelly.ListMethods", false, &res); err != nil {
		return nil, err
	}
	if !slices.Contains(res.Methods, "Temperature.GetStatus") {
		return nil, errors.New("Temperature.GetStatus method not available")
	}

	return v, nil
}

// Soc implements the api.Vehicle interface
func (v *Shelly) Soc() (float64, error) {
	var res Gen2Temperature
	err := v.conn.ExecCmd("Temperature.GetStatus", false, &res)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, res.Message)
	}
	return (res.TC / v.targetTC) * 100, err
}
