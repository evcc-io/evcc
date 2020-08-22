package charger

import (
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/grid-x/modbus"
)

const (
	wbSlaveID = 255

	wbRegStatus        = 100 // Input
	wbRegChargeTime    = 102 // Input
	wbRegActualCurrent = 300 // Holding
	wbRegEnable        = 400 // Coil
	wbRegMaxCurrent    = 528 // Holding

	timeout         = 1 * time.Second
	protocolTimeout = 2 * time.Second
)

// Wallbe is an api.ChargeController implementation for Wallbe wallboxes.
// It supports both wallbe controllers (post 2019 models) and older ones using the
// Phoenix EV-CC-AC1-M3-CBC-RCM-ETH controller.
// It uses Modbus TCP to communicate with the wallbox at modbus client id 255.
type Wallbe struct {
	log     *util.Logger
	client  modbus.Client
	handler *modbus.TCPClientHandler
	factor  int64
}

func init() {
	registry.Add("wallbe", NewWallbeFromConfig)
}

// NewWallbeFromConfig creates a Wallbe charger from generic config
func NewWallbeFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI    string
		Legacy bool
	}{
		URI: "192.168.0.8:502",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb := NewWallbe(cc.URI)

	if cc.Legacy {
		wb.factor = 1
	}

	return wb, nil
}

// NewWallbe creates a Wallbe charger
func NewWallbe(conn string) *Wallbe {
	handler := modbus.NewTCPClientHandler(conn)
	client := modbus.NewClient(handler)

	handler.SlaveID = wbSlaveID
	handler.Timeout = timeout
	handler.ProtocolRecoveryTimeout = protocolTimeout

	wb := &Wallbe{
		log:     util.NewLogger("wallbe"),
		client:  client,
		handler: handler,
		factor:  10,
	}

	return wb
}

// Status implements the Charger.Status interface
func (wb *Wallbe) Status() (api.ChargeStatus, error) {
	b, err := wb.client.ReadInputRegisters(wbRegStatus, 1)
	wb.log.TRACE.Printf("read status (%d): %0 X", wbRegStatus, b)
	if err != nil {
		wb.handler.Close()
		return api.StatusNone, err
	}

	return api.ChargeStatus(string(b[1])), nil
}

// Enabled implements the Charger.Enabled interface
func (wb *Wallbe) Enabled() (bool, error) {
	b, err := wb.client.ReadCoils(wbRegEnable, 1)
	wb.log.TRACE.Printf("read charge enable (%d): %0 X", wbRegEnable, b)
	if err != nil {
		wb.handler.Close()
		return false, err
	}

	return b[0] == 1, nil
}

// Enable implements the Charger.Enable interface
func (wb *Wallbe) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 0xFF00
	}

	b, err := wb.client.WriteSingleCoil(wbRegEnable, u)
	wb.log.TRACE.Printf("write charge enable %d %0X: %0 X", wbRegEnable, u, b)
	if err != nil {
		wb.handler.Close()
	}

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (wb *Wallbe) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	u := uint16(current * wb.factor)

	b, err := wb.client.WriteSingleRegister(wbRegMaxCurrent, u)
	wb.log.TRACE.Printf("write max current %d %0X: %0 X", wbRegMaxCurrent, u, b)
	if err != nil {
		wb.handler.Close()
	}

	return err
}

// ChargingTime yields current charge run duration
func (wb *Wallbe) ChargingTime() (time.Duration, error) {
	b, err := wb.client.ReadInputRegisters(wbRegChargeTime, 2)
	wb.log.TRACE.Printf("read charge time (%d): %0 X", wbRegChargeTime, b)
	if err != nil {
		wb.handler.Close()
		return 0, err
	}

	// 2 words, least significant word first
	secs := uint64(b[3])<<16 | uint64(b[2])<<24 | uint64(b[1]) | uint64(b[0])<<8
	return time.Duration(time.Duration(secs) * time.Second), nil
}
