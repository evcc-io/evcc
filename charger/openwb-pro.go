package charger

import (
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

func init() {
	registry.Add("openwbpro", NewOpenWBProFromConfig)
}

// https://openwb.de/main/?page_id=771

type openwbProStatus struct {
	Date           string
	Timestamp      int64
	Powers         []float64
	PowerAll       float64 `json:"power_all"`
	Currents       []float64
	Imported       float64
	Exported       float64
	PlugState      bool   `json:"plug_state"`
	ChargeState    bool   `json:"charge_state"`
	PhasesActual   int    `json:"phases_actual"`
	PhasesTarget   int    `json:"phases_target"`
	PhasesInUse    int    `json:"phases_in_use"`
	OfferedCurrent int    `json:"offered_current"`
	EvseSignaling  string `json:"evse_signaling"`
	Serial         string
}

// OpenWBPro charger implementation
type OpenWBPro struct {
	*request.Helper
	uri     string
	current float64
}

// NewOpenWBProFromConfig creates a OpenWBPro charger from generic config
func NewOpenWBProFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewOpenWBPro(util.DefaultScheme(cc.URI, "http"))
}

// NewOpenWBPro creates OpenWBPro charger
func NewOpenWBPro(uri string) (*OpenWBPro, error) {
	log := util.NewLogger("openwbpro")

	wb := &OpenWBPro{
		Helper:  request.NewHelper(log),
		uri:     strings.TrimRight(uri, "/"),
		current: 6, // 6A defined value
	}

	go wb.hearbeat(log)

	return wb, nil
}

func (wb *OpenWBPro) hearbeat(log *util.Logger) {
	for range time.NewTicker(30 * time.Second).C {
		if _, err := wb.get(); err != nil {
			log.ERROR.Printf("heartbeat: %v", err)
		}
	}
}

func (wb *OpenWBPro) get() (openwbProStatus, error) {
	var res openwbProStatus
	uri := fmt.Sprintf("%s/%s", wb.uri, "connect.php")
	err := wb.GetJSON(uri, &res)
	return res, err
}

func (wb *OpenWBPro) set(payload string) error {
	uri := fmt.Sprintf("%s/%s", wb.uri, "connect.php")
	resp, err := wb.Post(uri, "text/plain", strings.NewReader(payload))
	if err == nil {
		resp.Body.Close()
	}
	return err
}

// Status implements the api.Charger interface
func (wb *OpenWBPro) Status() (api.ChargeStatus, error) {
	resp, err := wb.get()
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
	res, err := wb.get()
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
	res, err := wb.get()
	return res.PowerAll, err
}

var _ api.MeterEnergy = (*OpenWBPro)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *OpenWBPro) TotalEnergy() (float64, error) {
	res, err := wb.get()
	return res.Imported / 1e3, err
}

var _ api.MeterCurrent = (*OpenWBPro)(nil)

// Currents implements the api.MeterCurrents interface
func (wb *OpenWBPro) Currents() (float64, float64, float64, error) {
	res, err := wb.get()
	if err != nil {
		return 0, 0, 0, err
	}

	if len(res.Currents) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid currents: %v", res.Currents)
	}

	return res.Currents[0], res.Currents[1], res.Currents[2], err
}

var _ api.ChargePhases = (*OpenWBPro)(nil)

// Phases1p3p implements the api.ChargePhases interface
func (wb *OpenWBPro) Phases1p3p(phases int) error {
	return wb.set(fmt.Sprintf("phasetarget=%d", phases))
}
