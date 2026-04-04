package charger

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Askoheat charger implementation for Askoma ASKOHEAT+ power-to-heat device
type Askoheat struct {
	*embed
	*request.Helper
	log      *util.Logger
	lp       loadpoint.API
	uri      string
	power    uint32 // atomic: last setpoint in watts
	enabled  bool
	sensor   int // temp sensor index 0-5
	maxPower int // from MODBUS_PAR_MAX_POWER
	minPower int // from MODBUS_PAR_HEATER1_POWER
	emaG     util.Cacheable[askoheatEMAResponse]
	conG     util.Cacheable[askoheatCONResponse]
}

type askoheatEMAResponse struct {
	Info           askoheatInfo `json:"ASKOHEAT_PLUS_INFO"`
	Status         string       `json:"MODBUS_EMA_STATUS"`
	HeaterLoad     string       `json:"MODBUS_EMA_HEATER_LOAD"`
	SetHeaterStep  string       `json:"MODBUS_EMA_SET_HEATER_STEP"`
	LoadSetpoint   string       `json:"MODBUS_EMA_LOAD_SETPOINT_VALUE"`
	TempSensor0    string       `json:"MODBUS_EMA_TEMPERATURE_FLOAT_SENSOR0"`
	TempSensor1    string       `json:"MODBUS_EMA_TEMPERATURE_FLOAT_SENSOR1"`
	TempSensor2    string       `json:"MODBUS_EMA_TEMPERATURE_FLOAT_SENSOR2"`
	TempSensor3    string       `json:"MODBUS_EMA_TEMPERATURE_FLOAT_SENSOR3"`
	TempSensor4    string       `json:"MODBUS_EMA_TEMPERATURE_FLOAT_SENSOR4"`
	TempSensor5    string       `json:"MODBUS_EMA_TEMPERATURE_FLOAT_SENSOR5"`
	StatusExtended string       `json:"MODBUS_EMA_STATUS_EXTENDED"`
}

type askoheatInfo struct {
	ArticleName     string `json:"ARTICLE_NAME"`
	SoftwareVersion string `json:"SOFTWARE_VERSION"`
	DeviceID        string `json:"DEVICEID"`
}

type askoheatPARResponse struct {
	MaxPower       string `json:"MODBUS_PAR_MAX_POWER"`
	Heater1Power   string `json:"MODBUS_PAR_HEATER1_POWER"`
	NumberOfHeater string `json:"MODBUS_PAR_NUMBER_OF_HEATER"`
	ArticleName    string `json:"MODBUS_PAR_ARTICLE_NAME"`
}

type askoheatCONResponse struct {
	TempLoadSetpoint string `json:"MODBUS_CON_TEMPERATURE_LOAD_SETPOINT"`
}

func init() {
	registry.AddCtx("askoheat", NewAskoheatFromConfig)
}

// NewAskoheatFromConfig creates an Askoheat charger from generic config
func NewAskoheatFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		embed      `mapstructure:",squash"`
		URI        string
		TempSensor int
		Cache      time.Duration
	}{
		embed: embed{
			Icon_:     "waterheater",
			Features_: []api.Feature{api.Continuous, api.IntegratedDevice, api.Heating},
		},
		TempSensor: 0,
		Cache:      5 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewAskoheat(ctx, &cc.embed, cc.URI, cc.TempSensor, cc.Cache)
}

// NewAskoheat creates an Askoma ASKOHEAT+ charger
func NewAskoheat(ctx context.Context, embed *embed, uri string, sensor int, cache time.Duration) (api.Charger, error) {
	if sensor < 0 || sensor > 5 {
		return nil, fmt.Errorf("invalid temp sensor: %d (must be 0-5)", sensor)
	}

	log := util.NewLogger("askoheat")

	wb := &Askoheat{
		embed:  embed,
		Helper: request.NewHelper(log),
		log:    log,
		uri:    util.DefaultScheme(uri, "http"),
		sensor: sensor,
	}

	// disable keep-alive to prevent EOF from stale connections
	wb.Client.Transport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: -1, // disable
		}).DialContext,
		DisableKeepAlives: true,
	}

	// cache EMA and CON responses to reduce HTTP requests per polling cycle
	wb.emaG = util.ResettableCached(func() (askoheatEMAResponse, error) {
		var res askoheatEMAResponse
		err := wb.GetJSON(wb.uri+"/GETEMA.JSON", &res)
		return res, err
	}, cache)

	wb.conG = util.ResettableCached(func() (askoheatCONResponse, error) {
		var res askoheatCONResponse
		err := wb.GetJSON(wb.uri+"/GETCON.JSON", &res)
		return res, err
	}, cache)

	// read device parameters
	par, err := wb.getPAR()
	if err != nil {
		return nil, fmt.Errorf("failed to read device parameters: %w", err)
	}

	wb.maxPower, err = strconv.Atoi(par.MaxPower)
	if err != nil {
		return nil, fmt.Errorf("invalid max power: %w", err)
	}

	wb.minPower, err = strconv.Atoi(par.Heater1Power)
	if err != nil {
		return nil, fmt.Errorf("invalid heater1 power: %w", err)
	}

	log.DEBUG.Printf("device: %s, max power: %d W, min power: %d W, heaters: %s",
		par.ArticleName, wb.maxPower, wb.minPower, par.NumberOfHeater)

	// check configured sensor
	ema, err := wb.emaG.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to read device status: %w", err)
	}

	if temp, err := wb.tempSensor(&ema); err != nil {
		log.WARN.Printf("temp sensor %d: %v (value: %.2f)", sensor, err, temp)
	}

	go wb.heartbeat(ctx, 30*time.Second)

	return wb, nil
}

func (wb *Askoheat) heartbeat(ctx context.Context, timeout time.Duration) {
	for tick := time.Tick(timeout); ; {
		select {
		case <-tick:
		case <-ctx.Done():
			return
		}

		if power := atomic.LoadUint32(&wb.power); power > 0 {
			if wb.enabled {
				if err := wb.setPower(int(power)); err != nil {
					wb.log.ERROR.Println("heartbeat:", err)
				}
			}
		}
	}
}

// HTTP helpers

func (wb *Askoheat) getPAR() (askoheatPARResponse, error) {
	var res askoheatPARResponse
	err := wb.GetJSON(wb.uri+"/GETPAR.JSON", &res)
	return res, err
}

func (wb *Askoheat) patchEMA(data map[string]string) error {
	req, err := request.New(http.MethodPatch, wb.uri+"/GETEMA.JSON", request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	_, err = wb.DoBody(req)
	return err
}

// tempSensor returns the temperature for the configured sensor index
func (wb *Askoheat) tempSensor(res *askoheatEMAResponse) (float64, error) {
	var raw string
	switch wb.sensor {
	case 0:
		raw = res.TempSensor0
	case 1:
		raw = res.TempSensor1
	case 2:
		raw = res.TempSensor2
	case 3:
		raw = res.TempSensor3
	case 4:
		raw = res.TempSensor4
	case 5:
		raw = res.TempSensor5
	}

	temp, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid temperature: %w", err)
	}

	if math.Abs(temp-9999.90) < 0.01 {
		return temp, fmt.Errorf("sensor %d disconnected (sentinel value %.2f)", wb.sensor, temp)
	}

	return temp, nil
}

func (wb *Askoheat) setPower(power int) error {
	return wb.patchEMA(map[string]string{
		"MODBUS_EMA_LOAD_SETPOINT_VALUE": strconv.Itoa(power),
	})
}

// Status implements the api.Charger interface
func (wb *Askoheat) Status() (api.ChargeStatus, error) {
	res, err := wb.emaG.Get()
	if err != nil {
		return api.StatusNone, err
	}

	status, err := strconv.ParseUint(res.Status, 10, 16)
	if err != nil {
		return api.StatusNone, fmt.Errorf("invalid status: %w", err)
	}

	// low byte bits 0-2 represent heater relays
	if uint16(status)&0x07 != 0 {
		return api.StatusC, nil
	}

	return api.StatusB, nil
}

// Enabled implements the api.Charger interface
func (wb *Askoheat) Enabled() (bool, error) {
	return wb.enabled, nil
}

// Enable implements the api.Charger interface
func (wb *Askoheat) Enable(enable bool) error {
	if !enable {
		if err := wb.patchEMA(map[string]string{
			"MODBUS_EMA_LOAD_SETPOINT_VALUE": "0",
			"MODBUS_EMA_SET_HEATER_STEP":     "0",
		}); err != nil {
			return err
		}
	}

	// power is set by MaxCurrentMillis; no PATCH needed for enable
	wb.enabled = enable
	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *Askoheat) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Askoheat)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Askoheat) MaxCurrentMillis(current float64) error {
	phases := 1
	if wb.lp != nil {
		if p := wb.lp.GetPhases(); p != 0 {
			phases = p
		}
	}

	power := int(voltage * current * float64(phases))

	// clamp to valid range: round up to minPower, cap at maxPower
	if power > 0 && power < wb.minPower {
		power = wb.minPower
	}
	if power > wb.maxPower {
		power = wb.maxPower
	}

	err := wb.setPower(power)
	if err == nil {
		atomic.StoreUint32(&wb.power, uint32(power))
	}

	return err
}

var _ api.Meter = (*Askoheat)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Askoheat) CurrentPower() (float64, error) {
	res, err := wb.emaG.Get()
	if err != nil {
		return 0, err
	}

	return strconv.ParseFloat(res.HeaterLoad, 64)
}

var _ api.Battery = (*Askoheat)(nil)

// Soc implements the api.Battery interface (returns temperature from configured sensor)
func (wb *Askoheat) Soc() (float64, error) {
	res, err := wb.emaG.Get()
	if err != nil {
		return 0, err
	}

	return wb.tempSensor(&res)
}

var _ api.SocLimiter = (*Askoheat)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (wb *Askoheat) GetLimitSoc() (int64, error) {
	res, err := wb.conG.Get()
	if err != nil {
		return 0, err
	}

	return strconv.ParseInt(res.TempLoadSetpoint, 10, 64)
}

var _ loadpoint.Controller = (*Askoheat)(nil)

// LoadpointControl implements loadpoint.Controller
func (wb *Askoheat) LoadpointControl(lp loadpoint.API) {
	wb.lp = lp
}
