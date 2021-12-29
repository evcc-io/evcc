package charger

import (
	"errors"
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
	uri string
}

func init() {
	registry.Add("garo", NewPCElectricFromConfig)
	registry.Add("pcelectric", NewPCElectricFromConfig)
}

// go:generate go run ../cmd/tools/decorate.go -f decorateEVSE -b *PCElectric -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.MeterCurrent,Currents,func() (float64, float64, float64, error)" -t "api.ChargerEx,MaxCurrentMillis,func(current float64) error" -t "api.Identifier,Identify,func() (string, error)"

// NewPCElectricFromConfig creates a PCElectric charger from generic config
func NewPCElectricFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewPCElectric(util.DefaultScheme(cc.URI, "http"))
	if err != nil {
		return wb, err
	}

	// return decorateEVSE(wb, currentPower, totalEnergy, currents, maxCurrentEx, identify), nil

	return wb, err
}

// NewPCElectric creates PCElectric charger
func NewPCElectric(uri string) (*PCElectric, error) {
	log := util.NewLogger("pce")

	wb := &PCElectric{
		Helper: request.NewHelper(log),
		uri:    strings.TrimRight(uri, "/") + "/servlet/rest/chargebox",
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *PCElectric) Status() (api.ChargeStatus, error) {
	var status pcelectric.Status

	uri := fmt.Sprintf("%s/%s", wb.uri, "status")
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
	uri := fmt.Sprintf("%s/%s", wb.uri, "status")
	err := wb.GetJSON(uri, &res)
	return res.Mode == "ALWAYS_ON", err
}

// Enable implements the api.Charger interface
func (wb *PCElectric) Enable(enable bool) error {
	return errors.New("not implemented")
}

// MaxCurrent implements the api.Charger interface
func (wb *PCElectric) MaxCurrent(current int64) error {
	uri := fmt.Sprintf("%s/%s", wb.uri, "currentlimit")

	data := fmt.Sprintf(`{
		"reducedIntervalsEnabled": true,
		"reducedCurrentIntervals": [
			{
				"schemaId": 1,
				"start": "00:00:00",
				"stop": "24:00:00",
				"weekday": 8,
				"chargeLimit": "%d"
			}
		]}`, current)
	req, err := request.New(http.MethodPost, uri, strings.NewReader(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	return wb.DoJSON(req, nil)
}

// // CurrentPower implements the api.Meter interface
// func (wb *PCElectric) currentPower() (float64, error) {
// }

// // TotalEnergy implements the api.MeterEnergy interface
// func (wb *PCElectric) totalEnergy() (float64, error) {
// }

// // Currents implements the api.MeterCurrents interface
// func (wb *PCElectric) currents() (float64, float64, float64, error) {
// }
