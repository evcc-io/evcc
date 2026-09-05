package devicehost

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util/templates"
)

type meter struct {
	*device
	implement.Caps
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

func (m *meter) CurrentPower() (float64, error) {
	var power float64
	err := call(m.device, api.Meter.CurrentPower, nil, &power)
	return power, err
}
