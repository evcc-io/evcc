package bluelink_us

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type Provider struct {
	statusG   func() (VehicleStatus, error)
	positionG func() (PositionResponse, error)
	chargeS   func(bool) error
	wakeup    func() (VehicleStatus, error)
}

func NewProvider(api *API, cache time.Duration) *Provider {
	v := &Provider{
		wakeup: api.Status,
		chargeS: func(enable bool) error {
			if enable {
				return api.ChargeStart()
			}
			return api.ChargeStop()
		},
	}

	v.statusG = util.Cached(func() (VehicleStatus, error) {
		return api.Status()
	}, cache)

	v.positionG = util.Cached(func() (PositionResponse, error) {
		return api.Position()
	}, cache)

	return v
}

var _ api.Battery = (*Provider)(nil)

func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}
	return res.SoC()
}

var _ api.ChargeState = (*Provider)(nil)

func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.statusG()
	if err != nil {
		return api.StatusNone, err
	}
	return res.Status()
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.statusG()
	if err != nil {
		return time.Time{}, err
	}
	return res.FinishTime()
}

var _ api.VehicleRange = (*Provider)(nil)

func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}
	return res.Range()
}

var _ api.VehicleClimater = (*Provider)(nil)

func (v *Provider) Climater() (bool, error) {
	res, err := v.statusG()
	if err != nil {
		return false, err
	}
	return res.Climater()
}

var _ api.SocLimiter = (*Provider)(nil)

func (v *Provider) GetLimitSoc() (int64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}
	return res.GetLimitSoc()
}

var _ api.VehiclePosition = (*Provider)(nil)

func (v *Provider) Position() (float64, float64, error) {
	res, err := v.positionG()
	if err != nil {
		return 0, 0, err
	}
	return res.Coord.Lat, res.Coord.Lon, nil
}

var _ api.ChargeController = (*Provider)(nil)

func (v *Provider) ChargeEnable(enable bool) error {
	return v.chargeS(enable)
}

var _ api.Resurrector = (*Provider)(nil)

func (v *Provider) WakeUp() error {
	_, err := v.wakeup()
	return err
}
