package devicehost

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util/templates"
)

type charger struct {
	*device
	implement.Caps
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

func (c *charger) Status() (api.ChargeStatus, error) {
	var status api.ChargeStatus
	err := call[api.ChargeState](c.device, "Status", nil, &status)
	return status, err
}

func (c *charger) Enabled() (bool, error) {
	var enabled bool
	err := call[api.Charger](c.device, "Enabled", nil, &enabled)
	return enabled, err
}

func (c *charger) Enable(enable bool) error {
	return call[api.Charger](c.device, "Enable", []any{enable})
}

func (c *charger) MaxCurrent(current int64) error {
	return call[api.CurrentController](c.device, "MaxCurrent", []any{current})
}
