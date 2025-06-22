package charger

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/openwb/hw"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/stianeikeland/go-rpio/v4"
	"github.com/volkszaehler/mbmd/encoding"
)

// OpenWbNative charger implementation
type OpenWbNative struct {
	conn        *modbus.Connection
	current     uint16
	log         *util.Logger
	rfIdChannel chan string
	rfId        string
	cpWait      float64
	chargePoint int
}

const (
	owbhwRegCurrent  = 1000
	owbhwRegStatus   = 1002
	owbhwRegFirmware = 1005
	owbhwRegConfig   = 2005
)

func init() {
	registry.AddCtx("openwb-native", NewOpenWbNativeFromConfig)
}

//go:generate go tool decorate -o openwb-native_decorators_linux.go -f decorateOpenWbNative -b *OpenWbNative -r api.Charger -t "api.ChargerEx,MaxCurrentMillis,func(float64) error" -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.Identifier,Identify,func() (string, error)"

// NewOpenWbNativeFromConfig creates an OpenWbNative DIN charger from generic config
func NewOpenWbNativeFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Phases1p3p      bool
		RfId            bool
		CpWait          float64
		ChargePoint     int
		modbus.Settings `mapstructure:",squash"`
	}{
		Settings: modbus.Settings{
			Baudrate: 9600,
			Comset:   "8N1",
			ID:       1},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewOpenWbNative(ctx, cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.Protocol(), cc.ID, cc.Phases1p3p, cc.RfId, cc.CpWait, cc.ChargePoint)
}

// NewOpenWbNative creates OpenWbNative charger
func NewOpenWbNative(ctx context.Context, uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8, hasPhases1p3p bool, hasRfid bool, cpWait float64, chargePoint int) (api.Charger, error) {
	conn, err := modbus.NewConnection(ctx, uri, device, comset, baudrate, proto, slaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("openwb-native")

	conn.Logger(log.TRACE)
	conn.Delay(200 * time.Millisecond)

	wb := &OpenWbNative{
		conn:    conn,
		current: 6, // assume min current
		log:     log,
	}

	var (
		phases1p3p       func(int) error
		maxCurrentMillis func(float64) error
		identify         func() (string, error)
	)

	// check EVSE DIN firmware and check & configure features
	bFirmware, err := wb.conn.ReadHoldingRegisters(owbhwRegFirmware, 1)
	if err != nil {
		return nil, err
	}

	if encoding.Uint16(bFirmware) >= 17 {
		wb.log.INFO.Print("EVSE firmware supports mA granularity, thus enabling it.")
		bConfig, err := wb.conn.ReadHoldingRegisters(owbhwRegConfig, 1)
		if err != nil {
			return nil, err
		}

		config := encoding.Uint16(bConfig)

		b := make([]byte, 2)
		config |= 0x0080 // set milliAmps bit
		binary.BigEndian.PutUint16(b, config)
		if _, err := wb.conn.WriteMultipleRegisters(owbhwRegConfig, 1, b); err != nil {
			return nil, err
		}

		maxCurrentMillis = wb.maxCurrentMillis
	}

	// configure special external hardware features
	if hasPhases1p3p {
		phases1p3p = wb.phases1p3p
	}

	if hasRfid {
		rfIdChannel, _, err := hw.NewRFIDHandler(ctx, log)
		if err != nil {
			return nil, err
		}
		// TODO: cleanup channel on charger close?
		wb.rfIdChannel = rfIdChannel

		identify = wb.identify
	}

	wb.cpWait = cpWait

	if (chargePoint < 0) || (chargePoint > 1) {
		return nil, fmt.Errorf("invalid chargepoint value: %d", chargePoint)
	}
	wb.chargePoint = chargePoint

	return decorateOpenWbNative(wb, maxCurrentMillis, phases1p3p, identify), nil
}

// Status implements the api.Charger interface
func (wb *OpenWbNative) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(owbhwRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	s := binary.BigEndian.Uint16(b)
	switch s {
	case 1: // ready
		return api.StatusA, nil
	case 2: // EV is present
		return api.StatusB, nil
	case 3: // charging
		return api.StatusC, nil
	case 4: // charging with ventilation
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *OpenWbNative) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(owbhwRegCurrent, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *OpenWbNative) Enable(enable bool) error {
	b := make([]byte, 2)
	if enable {
		binary.BigEndian.PutUint16(b, wb.current)
	}

	_, err := wb.conn.WriteMultipleRegisters(owbhwRegCurrent, 1, b)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *OpenWbNative) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(current))

	_, err := wb.conn.WriteMultipleRegisters(owbhwRegCurrent, 1, b)
	if err == nil {
		wb.current = uint16(current)
	}

	return err
}

// maxCurrentMillis implements the api.ChargerEx interface
func (wb *OpenWbNative) maxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	return wb.MaxCurrent(int64(current * 100)) // 0.01A Steps
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *OpenWbNative) phases1p3p(phases int) error {
	return wb.GpioWorkerExecutor(func() { wb.GpioSwitchPhases(phases) })
}

var _ api.Resurrector = (*OpenWbNative)(nil)

// WakeUp implements the api.Resurrector interface
func (wb *OpenWbNative) WakeUp() error {
	return wb.GpioWorkerExecutor(wb.GpioWakeup)
}

func (wb *OpenWbNative) GpioWorkerExecutor(worker func()) error {
	if err := wb.Enable(false); err != nil {
		return err
	}

	if err := rpio.Open(); err != nil {
		return err
	}
	defer rpio.Close()

	worker()

	if err := wb.Enable(true); err != nil {
		return err
	}

	return nil
}

// Worker function to toggle the GPIOs to switch the phases
func (wb *OpenWbNative) GpioSwitchPhases(phases int) {
	pinGpioCP := rpio.Pin(hw.GPIO_CP[wb.chargePoint])
	pinGpioPhases := rpio.Pin(hw.GPIO_3P[wb.chargePoint])
	if phases == 1 {
		pinGpioPhases = rpio.Pin(hw.GPIO_1P[wb.chargePoint])
	}
	pinGpioCP.Output()
	pinGpioPhases.Output()

	time.Sleep(time.Second)
	pinGpioCP.High() // enable phases switch relay (NO), disconnect CP
	time.Sleep(time.Second * time.Duration(wb.cpWait/2.0))
	pinGpioPhases.High() // move latching relay to desired position
	time.Sleep(time.Second)
	pinGpioPhases.Low() // lock latching relay
	time.Sleep(time.Second * time.Duration(wb.cpWait/2.0))
	pinGpioCP.Low() // disable phase switching, reconnect CP
	time.Sleep(time.Second)
}

// Worker function to toggle the GPIOs for the CP signal
func (wb *OpenWbNative) GpioWakeup() {
	pinGpioCP := rpio.Pin(hw.GPIO_CP[wb.chargePoint])
	pinGpioCP.Output()

	pinGpioCP.High()
	time.Sleep(time.Second * time.Duration(wb.cpWait))
	pinGpioCP.Low()
}

// Identify implements the api.Identifier interface
func (wb *OpenWbNative) identify() (string, error) {
	for {
		select {
		case rfid := <-wb.rfIdChannel:
			wb.log.INFO.Printf("Read RFID \"%s\" from channel", rfid)
			wb.rfId = rfid
		default:
			wb.log.INFO.Println("Nothing left to read from channel")
			return wb.rfId, nil
		}
	}
}
