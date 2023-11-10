package sma

import (
	"maps"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"gitlab.com/bboehmke/sunny"
)

// Device holds information for a Device and provides interface to get values
type Device struct {
	*sunny.Device

	log    *util.Logger
	values *util.Monitor[map[sunny.ValueID]any]
	once   sync.Once
}

// Run starts the receive loop once per device
func (d *Device) Run() {
	d.once.Do(d.run)
}

func (d *Device) run() {
	for range time.Tick(5 * time.Second) {
		if err := d.UpdateValues(); err != nil {
			d.log.ERROR.Println(err)
		}
	}
}

func (d *Device) UpdateValues() error {
	res, err := d.Device.GetValues()
	if err == nil {
		current, _ := d.values.Get()
		if current == nil {
			current = res
		} else {
			maps.Copy(current, res)
		}
		d.values.Set(current)
	}

	return err
}

func (d *Device) Values() (map[sunny.ValueID]any, error) {
	res, err := d.values.Get()
	if err != nil {
		return nil, err
	}

	return maps.Clone(res), nil
}

func AsFloat(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	case nil:
		return 0
	default:
		util.NewLogger("sma").WARN.Printf("unknown value type: %T", value)
		return 0
	}
}
