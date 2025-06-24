package charger

import (
	"context"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/openwb/hw"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/stianeikeland/go-rpio/v4"
)

// OpenWbHw charger implementation
type OpenWbHw struct {
	conn        *modbus.Connection
	current     int64
	log         *util.Logger
	rfIdChannel chan string
	rfId        string
}

const (
	owbhwRegAmpsConfig    = 1000
	owbhwRegVehicleStatus = 1002
)

func init() {
	registry.AddCtx("openwbhw", NewOpenWbHwFromConfig)
}

//go:generate go tool decorate -f decorateOpenWbHw -b *OpenWbHw -r api.Charger -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.Identifier,Identify,func() (string, error)"

// NewOpenWbHwFromConfig creates an OpenWbHw DIN charger from generic config
func NewOpenWbHwFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Phases1p3p      bool
		RfId            bool
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

	wb, err := NewOpenWbHw(ctx, cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.Protocol(), cc.ID)
	if err != nil {
		return nil, err
	}

	var phases1p3p func(int) error
	if cc.Phases1p3p {
		phases1p3p = wb.phases1p3p
	}

	if cc.RfId {
		log := util.NewLogger("openwbhw")
		rfIdChannel, _, err := hw.NewRFIDHandler(ctx, log)
		if err != nil {
			return nil, err
		}
		// Optionally, defer cleanup or store it if needed for later use
		wb.rfIdChannel = rfIdChannel
	}

	var identify func() (string, error)
	if _, err := wb.identify(); err == nil {
		identify = wb.identify
	}

	return decorateOpenWbHw(wb, phases1p3p, identify), nil
}

// NewOpenWbHw creates OpenWbHw charger
func NewOpenWbHw(ctx context.Context, uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8) (*OpenWbHw, error) {
	log := util.NewLogger("openwbhw")

	conn, err := modbus.NewConnection(ctx, uri, device, comset, baudrate, proto, slaveID)
	if err != nil {
		return nil, err
	}

	conn.Logger(log.TRACE)
	conn.Delay(200 * time.Millisecond)

	wb := &OpenWbHw{
		conn:    conn,
		current: 6, // assume min current
		log:     log,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *OpenWbHw) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(owbhwRegVehicleStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch b[1] {
	case 1: // ready
		return api.StatusA, nil
	case 2: // EV is present
		return api.StatusB, nil
	case 3: // charging
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", b[1])
	}
}

// Enabled implements the api.Charger interface
func (wb *OpenWbHw) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(owbhwRegAmpsConfig, 1)
	if err != nil {
		return false, err
	}

	enabled := b[1] != 0
	if enabled {
		wb.current = int64(b[1])
	}

	return enabled, nil
}

// Enable implements the api.Charger interface
func (wb *OpenWbHw) Enable(enable bool) error {
	b := []byte{0, 0}

	if enable {
		b[1] = byte(wb.current)
	}

	_, err := wb.conn.WriteMultipleRegisters(owbhwRegAmpsConfig, 1, b)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *OpenWbHw) MaxCurrent(current int64) error {
	b := []byte{0, byte(current)}

	_, err := wb.conn.WriteMultipleRegisters(owbhwRegAmpsConfig, 1, b)
	if err == nil {
		wb.current = current
	}

	return err
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

	// according to EV40 specification the CP level has to be set to -12V for at least 3 seconds
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
