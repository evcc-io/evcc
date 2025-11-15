package charger

import (
	"context"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/openwb/native"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/stianeikeland/go-rpio/v4"
)

const minCpWaitTime time.Duration = 5 * time.Second

// OpenWbNative charger implementation
type OpenWbNative struct {
	api.Charger
	log         *util.Logger
	rfId        native.RfIdContainer
	cpWait      time.Duration
	connector   int
	chargeState api.ChargeStatus
}

func init() {
	registry.AddCtx("openwb-native", NewOpenWbNativeFromConfig)
}

//go:generate go tool decorate -o openwb-native_decorators_linux.go -f decorateOpenWbNative -b *OpenWbNative -r api.Charger -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.Identifier,Identify,func() (string, error)"

// NewOpenWbNativeFromConfig creates an OpenWbNative DIN charger from generic config
func NewOpenWbNativeFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		Phases1p3p      bool
		RfId            string
		CpWait          time.Duration
		Connector       int
		modbus.Settings `mapstructure:",squash"`
	}{
		Settings: modbus.Settings{
			Baudrate: 9600,
			Comset:   "8N1",
			ID:       1,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewOpenWbNative(ctx, cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.Protocol(), cc.ID, cc.Phases1p3p, cc.RfId, cc.CpWait, cc.Connector)
}

// NewOpenWbNative creates OpenWbNative charger
func NewOpenWbNative(ctx context.Context, uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8, hasPhases1p3p bool, rfIdVidPid string, cpWait time.Duration, connector int) (api.Charger, error) {
	if (connector < 1) || (connector > 2) {
		return nil, fmt.Errorf("invalid connector value: %d", connector)
	}
	if cpWait < minCpWaitTime {
		return nil, fmt.Errorf("invalid cpwait value: %s, needs to be greater %s", cpWait.String(), minCpWaitTime)
	}

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
		phases1p3p func(int) error
		identify   func() (string, error)
	)

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

	return decorateOpenWbNative(wb, phases1p3p, identify), nil
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
	return wb.gpioExecute(func() error {
		return wb.gpioSwitchPhases(phases)
	})
}

var _ api.Resurrector = (*OpenWbNative)(nil)

// WakeUp implements the api.Resurrector interface
func (wb *OpenWbNative) WakeUp() error {
	return wb.gpioExecute(wb.gpioWakeup)
}

func (wb *OpenWbNative) gpioExecute(worker func() error) error {
	if err := wb.Enable(false); err != nil {
		return err
	}

	if err := rpio.Open(); err != nil {
		return err
	}
	defer rpio.Close()

	if err := worker(); err != nil {
		return err
	}

	return wb.Enable(true)
}

// Worker function to toggle the GPIOs to switch the phases
func (wb *OpenWbNative) gpioSwitchPhases(phases int) error {
	pinGpioCP := rpio.Pin(native.ChargePoints[wb.connector-1].PIN_CP)
	pinGpioPhases := rpio.Pin(native.ChargePoints[wb.connector-1].PIN_3P)
	if phases == 1 {
		pinGpioPhases = rpio.Pin(native.ChargePoints[wb.connector-1].PIN_1P)
	}
	pinGpioCP.Output()
	pinGpioPhases.Output()

	time.Sleep(time.Second)
	pinGpioCP.High() // enable phases switch relay (NO), disconnect CP

	time.Sleep(time.Second)
	pinGpioPhases.High() // move latching relay to desired position

	time.Sleep(wb.cpWait / 2)
	pinGpioPhases.Low() // lock latching relay

	time.Sleep(wb.cpWait / 2)
	pinGpioCP.Low() // disable phase switching, reconnect CP

	time.Sleep(time.Second)
	return nil
}

// Worker function to toggle the GPIOs for the CP signal
func (wb *OpenWbNative) gpioWakeup() error {
	pinGpioCP := rpio.Pin(native.ChargePoints[wb.connector-1].PIN_CP)
	pinGpioCP.Output()

	pinGpioCP.High()
	time.Sleep(wb.cpWait)
	pinGpioCP.Low()
	return nil
}

// Identify implements the api.Identifier interface
func (wb *OpenWbNative) identify() (string, error) {
	return wb.rfId.Get(), nil
}
