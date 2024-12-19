package charger

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/openwb/pro"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

func init() {
	registry.AddCtx("openwbpro", NewOpenWBProFromConfig)
}

// https://openwb.de/main/?page_id=771

// OpenWBPro charger implementation
type OpenWBPro struct {
	*request.Helper
	uri     string
	current float64
	statusG provider.Cacheable[pro.Status]
}

// NewOpenWBProFromConfig creates a OpenWBPro charger from generic config
func NewOpenWBProFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI   string
		Cache time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewOpenWBPro(ctx, util.DefaultScheme(cc.URI, "http"), cc.Cache)
}

// NewOpenWBPro creates OpenWBPro charger
func NewOpenWBPro(ctx context.Context, uri string, cache time.Duration) (*OpenWBPro, error) {
	log := util.NewLogger("owbpro")

	wb := &OpenWBPro{
		Helper:  request.NewHelper(log),
		uri:     strings.TrimRight(uri, "/"),
		current: 6, // 6A defined value
	}

	wb.statusG = provider.ResettableCached(func() (pro.Status, error) {
		var res pro.Status
		uri := fmt.Sprintf("%s/%s", wb.uri, "connect.php")
		err := wb.GetJSON(uri, &res)
		return res, err
	}, cache)

	go wb.heartbeat(ctx, log)

	return wb, nil
}

func (wb *OpenWBPro) heartbeat(ctx context.Context, log *util.Logger) {
	for tick := time.Tick(30 * time.Second); ; {
		select {
		case <-tick:
		case <-ctx.Done():
			return
		}

		if _, err := wb.statusG.Get(); err != nil {
			log.ERROR.Printf("heartbeat: %v", err)
		}
	}
}

func (wb *OpenWBPro) set(payload string) error {
	uri := fmt.Sprintf("%s/%s", wb.uri, "connect.php")
	resp, err := wb.Post(uri, "application/x-www-form-urlencoded", strings.NewReader(payload))
	if err == nil {
		resp.Body.Close()
		wb.statusG.Reset()
	}

	return err
}

// Status implements the api.Charger interface
func (wb *OpenWBPro) Status() (api.ChargeStatus, error) {
	resp, err := wb.statusG.Get()
	if err != nil {
		return api.StatusNone, err
	}

	res := api.StatusA
	switch {
	case resp.ChargeState:
		res = api.StatusC
	case resp.PlugState:
		res = api.StatusB
	}

	return res, nil
}

// Enabled implements the api.Charger interface
func (wb *OpenWBPro) Enabled() (bool, error) {
	res, err := wb.statusG.Get()
	return res.OfferedCurrent > 0, err
}

// Enable implements the api.Charger interface
func (wb *OpenWBPro) Enable(enable bool) error {
	payload := "ampere=0"
	if enable {
		payload = fmt.Sprintf("ampere=%.1f", wb.current)
	}

	return wb.set(payload)
}

// MaxCurrent implements the api.Charger interface
func (wb *OpenWBPro) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*OpenWBPro)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *OpenWBPro) MaxCurrentMillis(current float64) error {
	err := wb.set(fmt.Sprintf("ampere=%.1f", current))
	if err == nil {
		wb.current = current
	}
	return err
}

var _ api.Meter = (*OpenWBPro)(nil)

// CurrentPower implements the api.Meter interface
func (wb *OpenWBPro) CurrentPower() (float64, error) {
	res, err := wb.statusG.Get()
	return res.PowerAll, err
}

var _ api.MeterEnergy = (*OpenWBPro)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *OpenWBPro) TotalEnergy() (float64, error) {
	res, err := wb.statusG.Get()
	return res.Imported / 1e3, err
}

// getPhaseValues returns phase values
func (wb *OpenWBPro) getPhaseValues(f func(pro.Status) []float64) (float64, float64, float64, error) {
	status, err := wb.statusG.Get()
	if err != nil {
		return 0, 0, 0, err
	}

	res := f(status)

	if len(res) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid phases: %v", res)
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseVoltages = (*OpenWBPro)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *OpenWBPro) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(func(s pro.Status) []float64 {
		return s.Voltages
	})
}

var _ api.PhaseCurrents = (*OpenWBPro)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *OpenWBPro) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(func(s pro.Status) []float64 {
		return s.Currents
	})
}

var _ api.Battery = (*OpenWBPro)(nil)

// Soc implements the api.Battery interface
func (wb *OpenWBPro) Soc() (float64, error) {
	res, err := wb.statusG.Get()
	if err != nil {
		return 0, err
	}

	if time.Since(time.Unix(res.SocTimestamp, 0)) > 5*time.Minute {
		return 0, api.ErrNotAvailable
	}

	return float64(res.Soc), nil
}

var _ api.PhaseSwitcher = (*OpenWBPro)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *OpenWBPro) Phases1p3p(phases int) error {
	return wb.set(fmt.Sprintf("phasetarget=%d", phases))
}

var _ api.Identifier = (*OpenWBPro)(nil)

// Identify implements the api.Identifier interface
func (wb *OpenWBPro) Identify() (string, error) {
	res, err := wb.statusG.Get()
	if err != nil || res.VehicleID == "--" {
		return "", err
	}

	if res.VehicleID != "" {
		return res.VehicleID, nil
	}

	return res.RfidTag, nil
}
