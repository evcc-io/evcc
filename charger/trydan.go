package charger

// https://v2charge.com/trydan/

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

/*
	{
	  "ID": "B0DVU7",
	  "ChargeState": 0,
	  "ReadyState": 0,
	  "ChargePower": 0,
	  "ChargeEnergy": 20.49,
	  "SlaveError": 4,
	  "ChargeTime": 29101,
	  "HousePower": 0,
	  "FVPower": 0,
	  "BatteryPower": 0,
	  "Paused": 0,
	  "Locked": 0,
	  "Timer": 0,
	  "Intensity": 16,
	  "Dynamic": 0,
	  "MinIntensity": 6,
	  "MaxIntensity": 16,
	  "PauseDynamic": 0,
	  "FirmwareVersion": "2.1.7",
	  "DynamicPowerMode": 0,
	  "ContractedPower": 4600
	}

class SlaveCommunicationState(IntEnum):

	"""Enum for Slave Communication State."""

	NO_ERROR = 0
	COMMUNICATION = 1
	READING = 2
	SLAVE = 3
	WAITING_WIFI = 4
	WAITING_COMMUNICATION = 5
	WRONG_IP = 6
	SLAVE_NOT_FOUND = 7
	WRONG_SLAVE = 8
	NO_RESPONSE = 9
	CLAMP_NOT_CONNECTED = 10
	# MODBUS_ERRORS
	ILLEGAL_FUNCTION = 21
	ILLEGAL_DATA_ADDRESS = 22
	ILLEGAL_DATA_VALUE = 23
	SERVER_DEVICE_FAILURE = 24
	ACKNOWLEDGE = 25
	SERVER_DEVICE_BUSY = 26
	NEGATIVE_ACKNOWLEDGE = 27
	MEMORY_PARITY_ERROR = 28
	GATEWAY_PATH_UNAVAILABLE = 30
	GATEWAY_TARGET_NO_RESP = 31
	SERVER_RTU_INACTIVE244_TIMEOUT = 32
	INVALID_SERVER = 245
	CRC_ERROR = 246
	FC_MISMATCH = 247
	SERVER_ID_MISMATCH = 248
	PACKET_LENGTH_ERROR = 249
	PARAMETER_COUNT_ERROR = 250
	PARAMETER_LIMIT_ERROR = 251
	REQUEST_QUEUE_FULL = 252
	ILLEGAL_IP_OR_PORT = 253
	IP_CONNECTION_FAILED = 254
	TCP_HEAD_MISMATCH = 255
	EMPTY_MESSAGE = 256
	UNDEFINED_ERROR = 257
*/
type RealTimeData struct {
	ID               string  `json:"ID"`
	ChargeState      int     `json:"ChargeState"`
	ReadyState       int     `json:"ReadyState"`
	ChargePower      float64 `json:"ChargePower"`
	ChargeEnergy     float64 `json:"ChargeEnergy"`
	SlaveError       int     `json:"SlaveError"`
	ChargeTime       int     `json:"ChargeTime"`
	HousePower       float64 `json:"HousePower"`
	FVPower          float64 `json:"FVPower"`
	BatteryPower     float64 `json:"BatteryPower"`
	Paused           int     `json:"Paused"`
	Locked           int     `json:"Locked"`
	Timer            int     `json:"Timer"`
	Intensity        int     `json:"Intensity"`
	Dynamic          int     `json:"Dynamic"`
	MinIntensity     int     `json:"MinIntensity"`
	MaxIntensity     int     `json:"MaxIntensity"`
	PauseDynamic     int     `json:"PauseDynamic"`
	FirmwareVersion  string  `json:"FirmwareVersion"`
	DynamicPowerMode int     `json:"DynamicPowerMode"`
	ContractedPower  int     `json:"ContractedPower"`
}

// Trydan charger implementation
type Trydan struct {
	log *util.Logger
	*request.Helper
	uri     string
	statusG provider.Cacheable[RealTimeData]
	current int
	enabled bool
}

func init() {
	registry.Add("trydan", NewTrydanFromConfig)
}

// NewTrydanFromConfig creates a Trydan charger from generic config
func NewTrydanFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI   string
		Cache time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	return NewTrydan(cc.URI, cc.Cache)
}

// NewTrydan creates Trydan charger
func NewTrydan(uri string, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("trydan")
	c := &Trydan{
		log:    log,
		Helper: request.NewHelper(log),
		uri:    util.DefaultScheme(strings.TrimSuffix(uri, "/"), "http"),
	}

	c.statusG = provider.ResettableCached(func() (RealTimeData, error) {
		var res RealTimeData
		uri := fmt.Sprintf("%s/RealTimeData", c.uri)
		//log.DEBUG.Printf("Trydan status request %s", uri)
		err := c.GetJSON(uri, &res)
		log.DEBUG.Printf("Trydan status response %#v", res)

		return res, err
	}, cache)

	return c, nil
}

// Status implements the api.Charger interface
func (t Trydan) Status() (api.ChargeStatus, error) {
	/*
		status:
		  source: http
		  uri: http://192.168.1.111/RealTimeData
		  jq: .ChargeState | if . == 0 then "A" elif . == 1 then "B" elif . == 2 then "C" else "F" end
		  # 0,1,2, (A, B, C)
	*/
	data, err := t.statusG.Get()
	if err != nil {
		return api.StatusNone, err
	}
	switch state := data.ChargeState; state {
	case 0:
		return api.StatusA, nil
	case 1:
		return api.StatusB, nil
	case 2:
		return api.StatusC, nil
	default:
		return api.StatusF, nil
	}
}

// Enabled implements the api.Charger interface
func (c Trydan) Enabled() (bool, error) {
	/*
		enabled:
		  source: http
		  uri: http://192.168.1.111/RealTimeData
		  jq: .Paused | if . == 0 then 1 else 0 end

	*/
	data, err := c.statusG.Get()
	ret := data.Paused == 0 && data.Locked == 0
	c.log.DEBUG.Printf("Trydan Enabled: %t Paused: %d Locked: %d", ret, data.Paused, data.Locked)
	return ret, err
}

func (c Trydan) Set(parameter string, value int) error {
	uri := fmt.Sprintf("%s/write/%s=%d", c.uri, parameter, value)
	res, err := c.GetBody(uri)
	if err == nil {
		resStr := string(res[:])
		if resStr != "OK" {
			err = fmt.Errorf("command failed: %s", res)
		}
	}
	return err
}

// Enable implements the api.Charger interface
func (c Trydan) Enable(enable bool) error {
	/*
	   enable:
	      source: http
	      uri: http://192.168.1.111/write/Locked={{if .enable}}0{{else}}1{{end}}
	*/
	var _enable = 1

	if enable {
		_enable = 0
	}
	c.log.DEBUG.Printf("Trydan Set Paused: %d", _enable)
	err := c.Set("Paused", _enable)
	if err != nil {
		return err
	}
	c.log.DEBUG.Printf("Trydan Set Locked: %d", _enable)
	err = c.Set("Locked", _enable)
	if err != nil {
		return err
	}
	c.enabled = enable
	c.log.DEBUG.Printf("Trydan Enable: %t", c.enabled)
	return nil
}

// MaxCurrent implements the api.Charger interface
func (c Trydan) MaxCurrent(current int64) error {
	/*
		maxcurrent:
		  source: http
		  uri: http://192.168.1.111/write/Intensity=${maxcurrent:%d}
	*/
	err := c.Set("Intensity", c.current)
	if err == nil {
		c.current = int(current)
	}
	return err
}

var _ api.ChargeRater = (*Trydan)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c Trydan) ChargedEnergy() (float64, error) {
	data, err := c.statusG.Get()
	return data.ChargeEnergy, err
}

var _ api.ChargeTimer = (*Trydan)(nil)

// ChargeDuration implements the api.ChargeTimer interface
func (c Trydan) ChargeDuration() (time.Duration, error) {
	data, err := c.statusG.Get()
	return time.Duration(data.ChargeTime) * time.Second, err
}

var _ api.Meter = (*Trydan)(nil)

// CurrentPower implements the api.Meter interface
func (c Trydan) CurrentPower() (float64, error) {
	data, err := c.statusG.Get()
	return data.ChargePower, err
}

var _ api.Diagnosis = (*Trydan)(nil)

// Diagnose implements the api.Diagnosis interface
func (c *Trydan) Diagnose() {
	data, err := c.statusG.Get()
	if err != nil {
		fmt.Printf("%#v", data)
	}
}
