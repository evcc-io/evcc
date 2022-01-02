package charger

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/pcelectric"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// PCElectric charger implementation
type PCElectric struct {
	*request.Helper
	uri     string
	meterID int
}

func init() {
	registry.Add("garo", NewPCElectricFromConfig)
	registry.Add("pcelectric", NewPCElectricFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decoratePCE -b *PCElectric -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterCurrent,Currents,func() (float64, float64, float64, error)"

// NewPCElectricFromConfig creates a PCElectric charger from generic config
func NewPCElectricFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI     string
		MeterID int
	}{
		MeterID: 100,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewPCElectric(util.DefaultScheme(cc.URI, "http"), cc.MeterID)
	if err == nil {
		var res pcelectric.MeterInfo
		if err := wb.GetJSON(fmt.Sprintf("%s/meterinfo/CENTRAL%d", wb.uri, wb.meterID), &res); err == nil && res.MeterSerial != "" {
			return decoratePCE(wb, wb.currentPower, wb.currents), nil
		}
	}

	return wb, err
}

// NewPCElectric creates PCElectric charger
func NewPCElectric(uri string, meterID int) (*PCElectric, error) {
	log := util.NewLogger("pce")

	wb := &PCElectric{
		Helper:  request.NewHelper(log),
		uri:     strings.TrimRight(uri, "/") + "/servlet/rest/chargebox",
		meterID: meterID,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *PCElectric) Status() (api.ChargeStatus, error) {
	var status pcelectric.Status

	uri := fmt.Sprintf("%s/status", wb.uri)
	if err := wb.GetJSON(uri, &status); err != nil {
		return api.StatusNone, err
	}

	res := api.StatusA
	switch status.Connector {
	case "CONNECTED":
		res = api.StatusB
	case "CHARGING":
		res = api.StatusC
	}

	return res, nil
}

// Enabled implements the api.Charger interface
func (wb *PCElectric) Enabled() (bool, error) {
	var res pcelectric.Status
	uri := fmt.Sprintf("%s/status", wb.uri)
	err := wb.GetJSON(uri, &res)
	if err == nil && res.PowerMode == "ON" {
		return true, err
	}
	return false, err
}

// Enable implements the api.Charger interface
func (wb *PCElectric) Enable(enable bool) error {
	mode := "ALWAYS_OFF"
	if enable {
		mode = "ALWAYS_ON"
	}

	uri := fmt.Sprintf("%s/mode/%s", wb.uri, mode)
	req, err := request.New(http.MethodPost, uri, nil, request.JSONEncoding)
	if err == nil {
		_, err = wb.DoBody(req)
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *PCElectric) MaxCurrent(current int64) error {
	uri := fmt.Sprintf("%s/currentlimit", wb.uri)
	data := pcelectric.ReducedIntervals{
		ReducedIntervalsEnabled: true,
		ReducedCurrentIntervals: []pcelectric.ReducedCurrentInterval{
			{
				SchemaId:    1,
				Start:       "00:00:00",
				Stop:        "24:00:00",
				Weekday:     8,
				ChargeLimit: int(current),
			},
		},
	}

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		_, err = wb.DoBody(req)
	}

	return err
}

// CurrentPower implements the api.Meter interface
func (wb *PCElectric) currentPower() (float64, error) {
	var res pcelectric.MeterInfo
	uri := fmt.Sprintf("%s/meterinfo/CENTRAL%d", wb.uri, wb.meterID)
	err := wb.GetJSON(uri, &res)
	return float64(res.ApparentPower) * 1000, err
}

// Currents implements the api.MeterCurrents interface
func (wb *PCElectric) currents() (float64, float64, float64, error) {
	var res pcelectric.MeterInfo
	uri := fmt.Sprintf("%s/meterinfo/CENTRAL%d", wb.uri, wb.meterID)
	err := wb.GetJSON(uri, &res)
	return float64(res.Phase1Current) / 10, float64(res.Phase2Current) / 10, float64(res.Phase3Current) / 10, err
}
