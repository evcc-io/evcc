// Copyright (c) 2021 steff393, MIT license

package charger

import (
	"errors"
	"fmt"
	"strings"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

const minTime = time.Second * 3 // 3s

// wbecStatusResponse is the API response if status not OK
type wbecStatusResponse struct {
	ChgStat int      `json:"ChgStat,uint16"`   // car status
	CurrLim int      `json:"currLim,uint16"`   // current [A]
	EnergyI float64  `json:"energyI,float64"`  // energy [Wh]
	CurrL1  int      `json:"currL1,uint16"`    // current [A]
	CurrL2  int      `json:"currL2,uint16"`    // current [A]
	CurrL3  int      `json:"currL3,uint16"`    // current [A]
	Power   int      `json:"power,uint16"`     // power [W]
}

type wbecStatusResponseAll struct {
	Box []wbecStatusResponse  `json:"box"`        // 
}

// wbec charger implementation
type wbec struct {
	*request.Helper
	uri     string
	boxId   int
	current int64
	updated time.Time
	cache   wbecStatusResponse
}

func init() {
	registry.Add("wbec", NewWbecFromConfig)
}

// NewWbecFromConfig creates a charger from generic config
func NewWbecFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI   string
		BusId string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("must have uri")
	}
	if cc.BusId == "" {
		return nil, errors.New("must have busId")
	}

	return NewWbec(cc.URI, cc.BusId)
}

// NewWbec creates wbec charger
func NewWbec(uri, id string) (api.Charger, error) {
	log := util.NewLogger("wbec")
	tempId, _ := strconv.Atoi(strings.TrimSpace(id))
	wb := &wbec{
		Helper:   request.NewHelper(log),
		uri:      strings.TrimRight(uri, "/"),
		boxId:    tempId -1,
		current:  0,
	}

	return wb, nil
}

func (wb *wbec) response(payload string) (wbecStatusResponse, error) {
	var all wbecStatusResponseAll
	var status wbecStatusResponse

	if time.Since(wb.updated) < minTime {
		return wb.cache, nil
	}
	
	// if cache is too old, then fetch new data
	url := fmt.Sprintf("%s/json", wb.uri)
	if payload != "" {
		url += "?box=" + string(wb.boxId) + "&" + payload
	}

	err := wb.GetJSON(url, &all)
	if err == nil {
		wb.updated = time.Now()
		status     = all.Box[wb.boxId]
		wb.cache   = status
	}	
	return status, err
}


// Status implements the api.Charger interface
func (wb *wbec) Status() (api.ChargeStatus, error) {
	status, err := wb.response("")
	if err != nil {
		return api.StatusNone, err
	}

	switch status.ChgStat {
	case 2, 3:
		return api.StatusA, nil
	case 4, 5:
		return api.StatusB, nil
	case 6, 7:
		return api.StatusC, nil
	case 9:
		return api.StatusE, nil
	default:
		return api.StatusNone, fmt.Errorf("car unknown result: %d", status.ChgStat)
	}
}

// Enabled implements the api.Charger interface
func (wb *wbec) Enabled() (bool, error) {
	if wb.current != 0 {
		return true, nil
	} 
	return false, nil
}

// Enable implements the api.Charger interface
func (wb *wbec) Enable(enable bool) error {
	_, err := wb.response(fmt.Sprintf("currLim=%d", wb.current * 10))
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *wbec) MaxCurrent(current int64) error {
	wb.current = current
	_, err := wb.response(fmt.Sprintf("currLim=%d", current * 10))
	return err
}

var _ api.Meter = (*wbec)(nil)

// CurrentPower implements the api.Meter interface
func (wb *wbec) CurrentPower() (float64, error) {
	status, err := wb.response("")
	return float64(status.Power), err
}

var _ api.ChargeRater = (*wbec)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *wbec) ChargedEnergy() (float64, error) {
	status, err := wb.response("")
	return float64(status.EnergyI), err
}

var _ api.MeterCurrent = (*wbec)(nil)

// Currents implements the api.MeterCurrent interface
func (wb *wbec) Currents() (float64, float64, float64, error) {
	status, err := wb.response("")
	return float64(status.CurrL1) / 10, float64(status.CurrL2) / 10, float64(status.CurrL3) / 10, err
}
