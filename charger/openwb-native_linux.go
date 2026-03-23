package charger

import (
	"context"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/openwb/native"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/warthog618/go-gpiocdev"
)

const minCpWaitTime time.Duration = 5 * time.Second

// openWbGpioLines holds GPIO lines for a single charge point
type openWbGpioLines struct {
	cp  *gpiocdev.Line
	ph1 *gpiocdev.Line
	ph3 *gpiocdev.Line
}

// OpenWbNative charger implementation
type OpenWbNative struct {
	api.Charger
	log         *util.Logger
	rfId        native.RfIdContainer
	cpWait      time.Duration
	connector   int
	chargeState api.ChargeStatus
	gpio        openWbGpioLines
}

// gpioAction defines a single GPIO pin operation with timing
type gpioAction struct {
	pin   func()
	delay time.Duration
}

func init() {
	registry.AddCtx("openwb-native", NewOpenWbNativeFromConfig)
}

//go:generate go tool decorate -o openwb-native_decorators_linux.go -f decorateOpenWbNative -b *OpenWbNative -r api.Charger -t api.ChargerEx,api.PhaseSwitcher,api.Identifier

// NewOpenWbNativeFromConfig creates an OpenWbNative charger from generic config
func NewOpenWbNativeFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		Phases1p3p      bool
		RfId            string
		CpWait          time.Duration
		Connector       int
		Chip            string
		modbus.Settings `mapstructure:",squash"`
	}{
		Chip: "gpiochip0",
		Settings: modbus.Settings{
			Baudrate: 9600,
			Comset:   "8N1",
			ID:       1,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if (cc.Connector < 1) || (cc.Connector > 2) {
		return nil, fmt.Errorf("invalid connector value: %d", cc.Connector)
	}
	if cc.CpWait < minCpWaitTime {
		return nil, fmt.Errorf("invalid cpwait value: %v, needs to be greater than %s", cc.CpWait, minCpWaitTime)
	}

	return NewOpenWbNative(ctx, cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.Protocol(), cc.ID, cc.Phases1p3p, cc.RfId, cc.CpWait, cc.Connector, cc.Chip)
}

// NewOpenWbNative creates OpenWbNative charger
func NewOpenWbNative(ctx context.Context, uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8, hasPhases1p3p bool, rfIdVidPid string, cpWait time.Duration, connector int, chip string) (api.Charger, error) {
	log := util.NewLogger("openwb-native")
	log.DEBUG.Printf("Creating OpenWB native with 3 phases %t, rfid %s, cpwait %s, connector %d", hasPhases1p3p, rfIdVidPid, cpWait.String(), connector)

	evse, err := NewEvseDIN(ctx, uri, device, comset, baudrate, proto, slaveID)
	if err != nil {
		return nil, err
	}

	wb := &OpenWbNative{
		Charger:     evse,
		log:         log,
		cpWait:      cpWait,
		connector:   connector,
		chargeState: api.StatusNone,
	}

	var (
		phases1p3p       func(int) error
		identify         func() (string, error)
		maxCurrentMillis func(float64) error
	)

	if ex, ok := evse.(api.ChargerEx); ok {
		maxCurrentMillis = ex.MaxCurrentMillis
	}

	// configure special external hardware features
	if hasPhases1p3p {
		phases1p3p = wb.phases1p3p
	}

	if rfIdVidPid != "" {
		err := native.NewRFIDHandler(rfIdVidPid, ctx, &wb.rfId, log)
		if err != nil {
			return nil, err
		}

		identify = wb.identify
	}

	// initialize GPIO lines and set pins to output
	pins := native.ChargePoints[connector-1]
	for _, gpioConfig := range []struct {
		dst **gpiocdev.Line
		pin int
	}{
		{&wb.gpio.cp, pins.PIN_CP},
		{&wb.gpio.ph1, pins.PIN_1P},
		{&wb.gpio.ph3, pins.PIN_3P},
	} {
		line, err := gpiocdev.RequestLine(chip, gpioConfig.pin, gpiocdev.AsOutput(0))
		if err != nil {
			return nil, fmt.Errorf("failed to open GPIO pin %d: %w", gpioConfig.pin, err)
		}
		*gpioConfig.dst = line
	}

	return decorateOpenWbNative(wb, maxCurrentMillis, phases1p3p, identify), nil
}

// Status implements the api.Charger interface
func (wb *OpenWbNative) Status() (api.ChargeStatus, error) {
	res, err := wb.Charger.Status()
	if err != nil {
		return api.StatusA, err
	}

	if wb.chargeState != api.StatusA && res == api.StatusA {
		// Status changed from connected/charging to not connected, discard rfid
		wb.rfId.Set("")
	}
	wb.chargeState = res

	return res, nil
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *OpenWbNative) phases1p3p(phases int) error {
	return wb.gpioSwitchPhases(phases)
}

var _ api.Resurrector = (*OpenWbNative)(nil)

// WakeUp implements the api.Resurrector interface
func (wb *OpenWbNative) WakeUp() error {
	return wb.runGpioSequence([]gpioAction{
		{pin: func() { wb.gpio.cp.SetValue(1) }, delay: wb.cpWait},
		{pin: func() { wb.gpio.cp.SetValue(0) }, delay: 0},
	})
}

// runGpioSequence executes a sequence of GPIO operations
func (wb *OpenWbNative) runGpioSequence(seq []gpioAction) error {
	if err := wb.Enable(false); err != nil {
		return err
	}

	for _, a := range seq {
		a.pin()
		if a.delay > 0 {
			time.Sleep(a.delay)
		}
	}

	return wb.Enable(true)
}

// gpioSwitchPhases toggles the GPIOs to switch between 1-phase and 3-phase charging
func (wb *OpenWbNative) gpioSwitchPhases(phases int) error {
	phLine := wb.gpio.ph3
	if phases == 1 {
		phLine = wb.gpio.ph1
	}

	return wb.runGpioSequence([]gpioAction{
		{pin: func() { wb.gpio.cp.SetValue(1) }, delay: time.Second}, // enable phases switch relay (NO), disconnect CP
		{pin: func() { phLine.SetValue(1) }, delay: wb.cpWait / 2},   // move latching relay to desired position
		{pin: func() { phLine.SetValue(0) }, delay: wb.cpWait / 2},   // lock latching relay
		{pin: func() { wb.gpio.cp.SetValue(0) }, delay: time.Second}, // disable phase switching, reconnect CP
	})
}

// Identify implements the api.Identifier interface
func (wb *OpenWbNative) identify() (string, error) {
	return wb.rfId.Get(), nil
}
