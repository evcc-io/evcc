package charger

import (
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
	registry.Add("openwbpro", NewOpenWBProFromConfig)
}

// https://openwb.de/main/?page_id=771

// OpenWBPro charger implementation
type OpenWBPro struct {
	*request.Helper
	uri         string
	current     float64
	statusCache provider.Cacheable[pro.Status]
}

// NewOpenWBProFromConfig creates a OpenWBPro charger from generic config
func NewOpenWBProFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI   string
		Cache time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewOpenWBPro(util.DefaultScheme(cc.URI, "http"), cc.Cache)
}

// NewOpenWBPro creates OpenWBPro charger
func NewOpenWBPro(uri string, cache time.Duration) (*OpenWBPro, error) {
	log := util.NewLogger("owbpro")

	wb := &OpenWBPro{
		Helper:  request.NewHelper(log),
		uri:     strings.TrimRight(uri, "/"),
		current: 6, // 6A defined value
	}

	wb.statusCache = provider.ResettableCached(func() (pro.Status, error) {
		var res pro.Status
		uri := fmt.Sprintf("%s/%s", wb.uri, "connect.php")
		err := wb.GetJSON(uri, &res)
		return res, err
	}, cache)

	go wb.heartbeat(log)

	return wb, nil
}

func (wb *OpenWBPro) heartbeat(log *util.Logger) {
	for range time.Tick(30 * time.Second) {
		if _, err := wb.statusCache.Get(); err != nil {
			log.ERROR.Printf("heartbeat: %v", err)
		}
	}
}

func (wb *OpenWBPro) set(payload string) error {
	uri := fmt.Sprintf("%s/%s", wb.uri, "connect.php")
	resp, err := wb.Post(uri, "application/x-www-form-urlencoded", strings.NewReader(payload))
	if err == nil {
		resp.Body.Close()
		wb.statusCache.Reset()
	}

	return err
}

// Status implements the api.Charger interface
func (wb *OpenWBPro) Status() (api.ChargeStatus, error) {
	resp, err := wb.statusCache.Get()
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
	res, err := wb.statusCache.Get()
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
	res, err := wb.statusCache.Get()
	return res.PowerAll, err
}

var _ api.MeterEnergy = (*OpenWBPro)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *OpenWBPro) TotalEnergy() (float64, error) {
	res, err := wb.statusCache.Get()
	return res.Imported / 1e3, err
}

var _ api.PhaseCurrents = (*OpenWBPro)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *OpenWBPro) Currents() (float64, float64, float64, error) {
	res, err := wb.statusCache.Get()
	if err != nil {
		return 0, 0, 0, err
	}

	if len(res.Currents) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid currents: %v", res.Currents)
	}

	return res.Currents[0], res.Currents[1], res.Currents[2], err
}

var _ api.Battery = (*OpenWBPro)(nil)

// Soc implements the api.Battery interface
func (wb *OpenWBPro) Soc() (float64, error) {
	res, err := wb.statusCache.Get()
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
	res, err := wb.statusCache.Get()
	if err != nil || res.VehicleID == "--" {
		return "", err
	}

	return res.VehicleID, err
}
