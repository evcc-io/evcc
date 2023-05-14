package charger

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Pantabox charger implementation
type Pantabox struct {
	*request.Helper
	uri string
}

func init() {
	registry.Add("pantabox", NewPantaboxFromConfig)
}

// NewPantaboxFromConfig creates a Pantabox charger from generic config
func NewPantaboxFromConfig(other map[string]interface{}) (api.Charger, error) {
	var cc struct {
		URI string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewPantabox(util.DefaultScheme(cc.URI, "http"))
}

// NewPantabox creates Pantabox charger
func NewPantabox(uri string) (*Pantabox, error) {
	log := util.NewLogger("pantabox")

	wb := &Pantabox{
		Helper: request.NewHelper(log),
		uri:    util.DefaultScheme(strings.TrimRight(uri, "/"), "http") + "/api",
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Pantabox) Status() (api.ChargeStatus, error) {
	var res struct {
		State string
	}

	if err := wb.GetJSON(wb.uri+"/charger/state", &res); err != nil {
		return api.StatusNone, err
	}

	switch res.State {
	case "A", "B", "C", "D", "E", "F":
		return api.ChargeStatus(res.State), nil
	default:
		return api.StatusNone, fmt.Errorf("invalid state: %s", res.State)
	}
}

// Enabled implements the api.Charger interface
func (wb *Pantabox) Enabled() (bool, error) {
	var res struct {
		Enabled int `json:",string"`
	}

	err := wb.GetJSON(wb.uri+"/charger/enabled", &res)
	return res.Enabled > 0, err
}

// Enable implements the api.Charger interface
func (wb *Pantabox) Enable(enable bool) error {
	resp, err := wb.Post(wb.uri+"/charger/enable", request.PlainContent, strings.NewReader(fmt.Sprintf("%t", enable)))
	if err == nil {
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			err = fmt.Errorf("status: %d", resp.StatusCode)
		}
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Pantabox) MaxCurrent(current int64) error {
	resp, err := wb.Post(wb.uri+"/charger/current", request.PlainContent, strings.NewReader(fmt.Sprintf("%d", current)))
	if err == nil {
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			err = fmt.Errorf("status: %d", resp.StatusCode)
		}
	}

	return err
}

var _ api.Meter = (*Pantabox)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Pantabox) CurrentPower() (float64, error) {
	var res struct {
		Power float64 `json:",string"`
	}

	err := wb.GetJSON(wb.uri+"/meter/power", &res)
	return res.Power * 1e3, err
}

var _ api.Diagnosis = (*Pantabox)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Pantabox) Diagnose() {
	var curr struct {
		MaxCurrent int `json:",string"`
	}
	if err := wb.GetJSON(wb.uri+"/charger/maxcurrent", &curr); err == nil {
		fmt.Printf("\tMax current:\t%dA\n", curr.MaxCurrent)
	}
}
