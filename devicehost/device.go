package devicehost

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/devicehost/proto/pb"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/templates"
)

type config struct {
	Host       string
	Device     string
	Properties map[string]string
}

// device is a handle to a device instantiated on a host
type device struct {
	host *Host
	id   string
	caps []string
}

// newDevice instantiates a device on its host
func newDevice(ctx context.Context, class templates.Class, other map[string]any) (*device, error) {
	var cc config
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	h, err := byName(cc.Host)
	if err != nil {
		return nil, err
	}

	res, err := h.client.New(ctx, &pb.NewRequest{
		DeviceClass: class.String(),
		Type:        cc.Device,
		Properties:  cc.Properties,
	})
	if err != nil {
		return nil, err
	}

	return &device{host: h, id: res.GetId(), caps: res.GetCapabilities()}, nil
}

// call invokes a capability method, decoding the reply into the ret pointers
func (d *device) call(capability, method string, args []any, ret ...any) error {
	encoded := make([][]byte, 0, len(args))
	for _, arg := range args {
		b, err := json.Marshal(arg)
		if err != nil {
			return err
		}
		encoded = append(encoded, b)
	}

	res, err := d.host.client.Call(context.TODO(), &pb.CallRequest{
		Id:         d.id,
		Capability: capability,
		Method:     method,
		Args:       encoded,
	})
	if err != nil {
		return err
	}

	if len(res.GetRet()) < len(ret) {
		return fmt.Errorf("%s.%s: expected %d values, got %d", capability, method, len(ret), len(res.GetRet()))
	}

	for i, v := range ret {
		if err := json.Unmarshal(res.GetRet()[i], v); err != nil {
			return fmt.Errorf("%s.%s: %w", capability, method, err)
		}
	}

	return nil
}

// decorate applies the capabilities the host reports
func (d *device) decorate(c implement.Caps) {
	for _, name := range d.caps {
		if f, ok := capTable[name]; ok {
			f(d, c)
		}
	}
}

// Meter creates a meter from a device host
func Meter(ctx context.Context, other map[string]any) (api.Meter, error) {
	d, err := newDevice(ctx, templates.Meter, other)
	if err != nil {
		return nil, err
	}

	res := &meter{device: d, Caps: implement.New()}
	d.decorate(res.Caps)

	return res, nil
}

type meter struct {
	*device
	implement.Caps
}

func (m *meter) CurrentPower() (float64, error) {
	var power float64
	err := m.call("api.Meter", "CurrentPower", nil, &power)
	return power, err
}

// Charger creates a charger from a device host
func Charger(ctx context.Context, other map[string]any) (api.Charger, error) {
	d, err := newDevice(ctx, templates.Charger, other)
	if err != nil {
		return nil, err
	}

	res := &charger{device: d, Caps: implement.New()}
	d.decorate(res.Caps)

	return res, nil
}

type charger struct {
	*device
	implement.Caps
}

func (c *charger) Status() (api.ChargeStatus, error) {
	var status api.ChargeStatus
	err := c.call("api.ChargeState", "Status", nil, &status)
	return status, err
}

func (c *charger) Enabled() (bool, error) {
	var enabled bool
	err := c.call("api.Charger", "Enabled", nil, &enabled)
	return enabled, err
}

func (c *charger) Enable(enable bool) error {
	return c.call("api.Charger", "Enable", []any{enable})
}

func (c *charger) MaxCurrent(current int64) error {
	return c.call("api.CurrentController", "MaxCurrent", []any{current})
}
