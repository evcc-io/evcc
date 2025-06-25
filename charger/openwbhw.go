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

// OpenWbHw charger implementation
type OpenWbHw struct {
	conn        *modbus.Connection
	current     uint16
	log         *util.Logger
	rfIdChannel chan string
	rfId        string
}

const (
	owbhwRegCurrent  = 1000
	owbhwRegStatus   = 1002
	owbhwRegFirmware = 1005
	owbhwRegConfig   = 2005
)

func init() {
	registry.AddCtx("openwbhw", NewOpenWbHwFromConfig)
}

//go:generate go tool decorate -f decorateOpenWbHw -b *OpenWbHw -r api.Charger -t "api.ChargerEx,MaxCurrentMillis,func(float64) error" -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.Identifier,Identify,func() (string, error)"

// NewOpenWbHwFromConfig creates an OpenWbHw DIN charger from generic config
func NewOpenWbHwFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Phases1p3p bool
		RfId       bool
		MilliAmps  bool
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewOpenWbHw(ctx, cc.Phases1p3p, cc.RfId, cc.MilliAmps)
}

// NewOpenWbHw creates OpenWbHw charger
func NewOpenWbHw(ctx context.Context, hasPhases1p3p bool, hasRfid bool, configureMilliAmps bool) (api.Charger, error) {
	conn, err := modbus.NewConnection(ctx, "", "/dev/ttyUSB0", "8N1", 9600, modbus.Rtu, 1)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("openwbhw")

	conn.Logger(log.TRACE)
	conn.Delay(200 * time.Millisecond)

	wb := &OpenWbHw{
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
		bConfig, err := wb.conn.ReadHoldingRegisters(owbhwRegConfig, 1)
		if err != nil {
			return nil, err
		}

		config := encoding.Uint16(bConfig)

		if configureMilliAmps != (config&0x80 != 0) {
			b := make([]byte, 2)
			if configureMilliAmps {
				config |= 0x0080 // set milliAmps bit
				binary.BigEndian.PutUint16(b, config)
			} else {
				config &= 0xff7F // clear milliAmps bit
				binary.BigEndian.PutUint16(b, config)
			}
			if _, err := wb.conn.WriteMultipleRegisters(owbhwRegConfig, 1, b); err != nil {
				return nil, err
			}
		}

		if config&0x80 != 0 {
			maxCurrentMillis = wb.maxCurrentMillis
		}
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

	return decorateOpenWbHw(wb, maxCurrentMillis, phases1p3p, identify), nil
}

// Status implements the api.Charger interface
func (wb *OpenWbHw) Status() (api.ChargeStatus, error) {
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
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *OpenWbHw) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(owbhwRegCurrent, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *OpenWbHw) Enable(enable bool) error {
	b := make([]byte, 2)
	if enable {
		binary.BigEndian.PutUint16(b, wb.current)
	}

	_, err := wb.conn.WriteMultipleRegisters(owbhwRegCurrent, 1, b)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *OpenWbHw) MaxCurrent(current int64) error {
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

// maxCurrentMillis implements the api.ChargerEx interface (Wallbe Firmware only)
func (wb *OpenWbHw) maxCurrentMillis(current float64) error {
	return wb.MaxCurrent(int64(current * 100)) // 0.01A Steps
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *OpenWbHw) phases1p3p(phases int) error {
	if err := wb.Enable(false); err != nil {
		return err
	}

	if err := rpio.Open(); err != nil {
		return err
	}
	defer rpio.Close()

	pinGpioCP := rpio.Pin(hw.GPIO_CP)
	pinGpioPhases := rpio.Pin(hw.GPIO_3P)
	if phases == 1 {
		pinGpioPhases = rpio.Pin(hw.GPIO_1P)
	}
	pinGpioCP.Output()
	pinGpioPhases.Output()

	time.Sleep(time.Second)
	pinGpioCP.High() // enable phases switch relay (NO), disconnect CP
	time.Sleep(time.Second)
	pinGpioPhases.High() // move latching relay to desired position
	time.Sleep(time.Second)
	pinGpioPhases.Low() // lock latching relay
	time.Sleep(time.Second)
	pinGpioCP.Low() // disable phase switching, reconnect CP
	time.Sleep(time.Second)

	if err := wb.Enable(true); err != nil {
		return err
	}

	return nil
}

var _ api.Resurrector = (*OpenWbHw)(nil)

// WakeUp implements the api.Resurrector interface
func (wb *OpenWbHw) WakeUp() error {
	if err := wb.Enable(false); err != nil {
		return err
	}

	if err := rpio.Open(); err != nil {
		return err
	}
	defer rpio.Close()

	gpioPinCP := rpio.Pin(hw.GPIO_CP)
	gpioPinCP.Output()

	gpioPinCP.High()
	time.Sleep(time.Second * time.Duration(3))
	gpioPinCP.Low()

	if err := wb.Enable(true); err != nil {
		return err
	}

	return nil
}

// Identify implements the api.Identifier interface
func (wb *OpenWbHw) identify() (string, error) {
	var completed bool = false

	for !completed {
		select {
		case rfid := <-wb.rfIdChannel:
			wb.log.INFO.Printf("Read RFID \"%s\" from channel", rfid)
			wb.rfId = rfid
		default:
			wb.log.INFO.Println("Nothing left to read from channel")
			completed = true
		}
	}

	return wb.rfId, nil
}
