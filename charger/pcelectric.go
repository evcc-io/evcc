package charger

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/pcelectric"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
)

// PCElectric charger implementation
type PCElectric struct {
	*request.Helper
	log *util.Logger

	uri        string // http://garo2216247:8080/servlet
	slaveIndex int    // 0 = Master, 1..n Slave
	meter      string // <CENTRAL100|CENTRAL101|INTERNAL|EXTERNAL|TWIN>

	lbmode       bool // true/false (wird automatisch bestimmt)
	serialNumber int  // 1234567
}

func init() {
	registry.Add("garo", NewPCElectricFromConfig)
	registry.Add("pcelectric", NewPCElectricFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decoratePCE -b *PCElectric -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)"

// NewPCElectricFromConfig creates a PCElectric charger from generic config
func NewPCElectricFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI        string
		SlaveIndex int
		Meter      string
	}{
		Meter: "CENTRAL100",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewPCElectric(util.DefaultScheme(cc.URI, "http"), cc.SlaveIndex, cc.Meter)
	if err == nil && wb.slaveIndex == 0 { // Nur Master hat den Zähler...leider
		var res pcelectric.MeterInfo
		if err := wb.GetJSON(wb.meter, &res); err == nil && res.MeterSerial != "" {
			return decoratePCE(wb, wb.currentPower, wb.totalEnergy, wb.currents), nil
		}

		wb.meter = ""
	}

	return wb, err
}

// NewPCElectric creates PCElectric charger
func NewPCElectric(uri string, slaveIndex int, meter string) (*PCElectric, error) {
	log := util.NewLogger("pce")
	uri = strings.TrimSuffix(strings.TrimRight(uri, "/"), "/servlet") + "/servlet/rest/chargebox"

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	wb := &PCElectric{
		Helper:     request.NewHelper(log),
		log:        log,
		uri:        uri,
		slaveIndex: slaveIndex,
		meter:      strings.ToUpper(meter),
	}

	// Nur Master: lb Config auslesen.
	// Ohne Loadbalancer: Steuerung über currentlimit
	// Mit Loadbalander: Steuerung über loadBalancingFuse
	var lbconfig pcelectric.LbConfig
	uri = fmt.Sprintf("%s/lbconfig/false", wb.uri)
	if err := wb.GetJSON(uri, &lbconfig); err == nil {
		wb.lbmode = lbconfig.MasterLoadBalanced
		wb.serialNumber = lbconfig.Slaves[wb.slaveIndex].SerialNumber
		log.DEBUG.Printf("lbmode: %t  serial: %d ", wb.lbmode, wb.serialNumber)
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *PCElectric) Status() (api.ChargeStatus, error) {
	var chargeStatus int
	var sessionStartTime int64

	if wb.slaveIndex == 0 {
		var status pcelectric.Status

		uri := fmt.Sprintf("%s/status", wb.uri)
		if err := wb.GetJSON(uri, &status); err != nil {
			return api.StatusNone, err
		}
		chargeStatus = status.ChargeStatus
		sessionStartTime = status.SessionStartTime
	} else {
		var status pcelectric.SlaveStatus

		uri := fmt.Sprintf("%s/slaves/false", wb.uri)
		if err := wb.GetJSON(uri, &status); err != nil {
			return api.StatusNone, err
		}
		if wb.slaveIndex >= len(status) {
			return api.StatusNone, nil
		}
		chargeStatus = status[wb.slaveIndex].ChargeStatus
		sessionStartTime = status[wb.slaveIndex].SessionStartTime
	}
	wb.log.DEBUG.Printf("chargeStatus: %d", chargeStatus)

	var res api.ChargeStatus
	switch chargeStatus {
	case 0x00, 0x10: // notconnected
		res = api.StatusA
	case 0x30: // connected
		res = api.StatusB
	case 0x40: // charging
		res = api.StatusC
	case 0x42, // chargepaused
		0x50, // chargefinished
		0x60: // chargecancelled
		res = api.StatusB
	case 0x90: // unavailable
		if sessionStartTime > 0 {
			res = api.StatusB
		} else {
			res = api.StatusF
		}
	case 0x95, // dcfault
		0x96, // dchardwarefault
		0x9A, // cpfault
		0x9B: // cpshorted
		res = api.StatusE
	case 0x70, // overheat
		0x80, // criticaltemperature
		0x91, // reserved
		0x9C, // remotedisabled
		0x9D, // dlmfault
		0xA0, // cablefault
		0xA1,
		0xA2, // lockingfault
		0xA3,
		0xA4, // contactorfault
		0xA8, // rcdfault
		0xF0, // wait
		0xF1: // ventfault
		res = api.StatusF
	default: // generalfault
		res = api.StatusF
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
	if wb.slaveIndex > 0 {
		return nil // Slave wird immer mit dem Master geschaltet!
	}

	// Master Only !!
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

func (wb *PCElectric) MinCurrent(current int64) error {
	data := pcelectric.MinCurrentLimitStruct{
		{
			MinCurrentLimit: int(current), // default=6
			SerialNumber:    wb.serialNumber,
			TwinSerial:      -1,
		},
	}

	uri := fmt.Sprintf("%s/mincurrentlimit", wb.uri)
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		_, err = wb.DoBody(req)
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *PCElectric) MaxCurrent(current int64) error {
	if wb.slaveIndex > 0 {
		return nil // Slave wird immer mit dem Master geschaltet!
	}

	// Ohne Loadbalancer Regelung über currentlimit:
	if !wb.lbmode {
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

		uri := fmt.Sprintf("%s/currentlimit", wb.uri)
		req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
		if err == nil {
			_, err = wb.DoBody(req)
		}

		return err
	}

	// Mit Loadbalancer Regelung über lbconfig/LoadBalancingFuse
	var data pcelectric.LbConfigShort
	uri := fmt.Sprintf("%s/lbconfig/false", wb.uri)
	if err := wb.GetJSON(uri, &data); err != nil {
		return err
	}

	data.LoadBalancingFuse = int(current)

	uri = fmt.Sprintf("%s/lbconfig", wb.uri)
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		_, err = wb.DoBody(req)
	}

	return err
}

// CurrentPower implements the api.Meter interface W
func (wb *PCElectric) currentPower() (float64, error) {
	l1, l2, l3, err := wb.currents()
	return 230 * (l1 + l2 + l3), err
}

// TotalEnergy implements the api.MeterEnergy interface kwh
func (wb *PCElectric) totalEnergy() (float64, error) {
	var res pcelectric.MeterInfo
	uri := fmt.Sprintf("%s/meterinfo/%s", wb.uri, wb.meter)
	err := wb.GetJSON(uri, &res)
	return float64(res.AccEnergy) / 1e3, err
}

// Currents implements the api.PhaseCurrentss interface A
func (wb *PCElectric) currents() (float64, float64, float64, error) {
	var res pcelectric.MeterInfo
	uri := fmt.Sprintf("%s/meterinfo/%s", wb.uri, wb.meter)
	err := wb.GetJSON(uri, &res)
	return float64(res.Phase1Current) / 10, float64(res.Phase2Current) / 10, float64(res.Phase3Current) / 10, err
}
