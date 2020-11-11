package vehicle

import (
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
)

type embed struct {
	title    string
	capacity int64
}

// Title implements the Vehicle.Title interface
func (m *embed) Title() string {
	return m.title
}

// Capacity implements the Vehicle.Capacity interface
func (m *embed) Capacity() int64 {
	return m.capacity
}

//go:generate go run ../cmd/tools/decorate.go -p vehicle -f decorateVehicle -b api.Vehicle -o vehicle_decorators -t "api.VehicleStatus,Status,func() (api.ChargeStatus, error)"

// Vehicle is an api.Vehicle implementation with configurable getters and setters.
type Vehicle struct {
	*embed
	chargeG func() (float64, error)
	statusG func() (string, error)
}

func init() {
	registry.Add("default", NewConfigurableFromConfig)
}

// NewConfigurableFromConfig creates a new Vehicle
func NewConfigurableFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title    string
		Capacity int64
		Charge   provider.Config
		Status   *provider.Config
		Cache    time.Duration
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	for k, v := range map[string]string{"charge": cc.Charge.Type} {
		if v == "" {
			return nil, fmt.Errorf("default vehicle config: %s required", k)
		}
	}

	getter, err := provider.NewFloatGetterFromConfig(cc.Charge)
	if err != nil {
		return nil, err
	}

	if cc.Cache > 0 {
		getter = provider.NewCached(getter, cc.Cache).FloatGetter()
	}

	v := &Vehicle{
		embed:   &embed{cc.Title, cc.Capacity},
		chargeG: getter,
	}

	// decorate vehicle with Status
	var status func() (api.ChargeStatus, error)
	if cc.Status != nil {
		v.statusG, err = provider.NewStringGetterFromConfig(*cc.Status)
		if err != nil {
			return nil, err
		}
		status = v.status
	}

	res := decorateVehicle(v, status)

	return res, nil
}

// ChargeState implements the Vehicle.ChargeState interface
func (m *Vehicle) ChargeState() (float64, error) {
	return m.chargeG()
}

// ChargeState implements the Vehicle.ChargeState interface
func (m *Vehicle) status() (api.ChargeStatus, error) {
	status := api.StatusF

	statusS, err := m.statusG()
	if err == nil {
		status = api.ChargeStatus(statusS)
	}

	return status, err
}
