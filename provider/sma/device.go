package sma

import (
	"fmt"
	"sync"
	"time"

	"dario.cat/mergo"
	"github.com/evcc-io/evcc/util"
	"gitlab.com/bboehmke/sunny"
)

// Device holds information for a Device and provides interface to get values
type Device struct {
	*sunny.Device

	log    *util.Logger
	mux    sync.Mutex
	wait   *util.Waiter
	values map[sunny.ValueID]interface{}
	once   sync.Once
}

// StartUpdateLoop if not already started
func (d *Device) StartUpdateLoop() {
	d.once.Do(func() {
		go func() {
			for range time.Tick(time.Second * 5) {
				if err := d.UpdateValues(); err != nil {
					d.log.ERROR.Println(err)
				}
			}
		}()
	})
}

func (d *Device) UpdateValues() error {
	d.mux.Lock()
	defer d.mux.Unlock()

	values, err := d.Device.GetValues()
	if err == nil {
		err = mergo.Merge(&d.values, values, mergo.WithOverride)
		d.wait.Update()
	}

	return err
}

func (d *Device) Values() (map[sunny.ValueID]interface{}, error) {
	// ensure update loop was started
	d.StartUpdateLoop()

	d.mux.Lock()
	defer d.mux.Unlock()

	if late := d.wait.Overdue(); late > 0 {
		return nil, fmt.Errorf("update timeout: %v", late.Truncate(time.Second))
	}

	// return a copy of the map to avoid race conditions
	values := make(map[sunny.ValueID]interface{}, len(d.values))
	for key, value := range d.values {
		values[key] = value
	}
	return values, nil
}

func AsFloat(value interface{}) float64 {
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
