package charger

import (
	"bytes"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/nrg/ble"
	"github.com/evcc-io/evcc/util"
	"github.com/godbus/dbus/v5"
	"github.com/lunixbochs/struc"
	"github.com/muka/go-bluetooth/bluez/profile/adapter"
	"github.com/muka/go-bluetooth/bluez/profile/agent"
	"github.com/muka/go-bluetooth/bluez/profile/device"
	"github.com/muka/go-bluetooth/hw"
)

const nrgTimeout = 10 * time.Second

// NRGKickBLE charger implementation
type NRGKickBLE struct {
	mu            sync.Mutex
	log           *util.Logger
	timer         *time.Ticker
	adapter       *adapter.Adapter1
	agent         *agent.SimpleAgent
	dev           *device.Device1
	device        string
	mac           string
	pin           int
	pauseCharging bool
	current       int
}

func init() {
	registry.Add("nrgkick-bluetooth", NewNRGKickBLEFromConfig)
}

// NewNRGKickBLEFromConfig creates a NRGKickBLE charger from generic config
func NewNRGKickBLEFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct{ Device, Mac, PIN string }{
		Device: "hci0",
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// decode PIN with leading zero
	pin, err := strconv.Atoi(cc.PIN)
	if err != nil {
		return nil, fmt.Errorf("invalid pin: %s", cc.PIN)
	}

	return NewNRGKickBLE(cc.Device, cc.Mac, pin)
}

// NewNRGKickBLE creates NRGKickBLE charger
func NewNRGKickBLE(device, mac string, pin int) (*NRGKickBLE, error) {
	logger := util.NewLogger("nrg-bt")

	ainfo, err := hw.GetAdapter(device)
	if err != nil {
		return nil, err
	}

	adapt, err := adapter.NewAdapter1FromAdapterID(ainfo.AdapterID)
	if err != nil {
		return nil, err
	}

	// Connect DBus System bus
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, err
	}

	// do not reuse agent0 from service
	agent.NextAgentPath()

	ag := agent.NewSimpleAgent()
	if err := agent.ExposeAgent(conn, ag, agent.CapNoInputNoOutput, true); err != nil {
		return nil, err
	}

	wb := &NRGKickBLE{
		log:     logger,
		timer:   time.NewTicker(2 * time.Second),
		device:  ainfo.AdapterID,
		mac:     mac,
		pin:     pin,
		adapter: adapt,
		agent:   ag,
	}

	return wb, nil
}

func (wb *NRGKickBLE) connect() (*device.Device1, error) {
	dev, err := ble.FindDevice(wb.adapter, wb.mac, nrgTimeout)
	if err != nil {
		return nil, fmt.Errorf("find device: %s", err)
	}

	err = ble.Connect(dev, wb.agent, wb.device)
	return dev, err
}

func (wb *NRGKickBLE) close() {
	if wb.dev != nil {
		wb.dev.Close()
		wb.dev = nil
	}
}

func (wb *NRGKickBLE) read(service string, res interface{}) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	<-wb.timer.C

	if wb.dev == nil {
		dev, err := wb.connect()
		if err != nil {
			return err
		}
		wb.dev = dev
	}

	char, err := wb.dev.GetCharByUUID(service)
	if err != nil {
		wb.close()
		return err
	}

	b, err := char.ReadValue(map[string]interface{}{})
	if err != nil {
		wb.close()
		return err
	}
	wb.log.TRACE.Printf("read %s %0x", service, b)

	return struc.Unpack(bytes.NewReader(b), res)
}

func (wb *NRGKickBLE) write(service string, val interface{}) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	var out bytes.Buffer
	if err := struc.Pack(&out, val); err != nil {
		return err
	}
	wb.log.TRACE.Printf("write %s %0x", service, out.Bytes())

	<-wb.timer.C

	if wb.dev == nil {
		dev, err := wb.connect()
		if err != nil {
			return err
		}
		wb.dev = dev
	}

	char, err := wb.dev.GetCharByUUID(service)
	if err != nil {
		wb.close()
		return err
	}

	if err := char.WriteValue(out.Bytes(), map[string]interface{}{}); err != nil {
		wb.close()
		return err
	}

	return nil
}

func (wb *NRGKickBLE) mergeSettings(info ble.Info) ble.Settings {
	return ble.Settings{
		PIN:                  wb.pin,
		ChargingEnergyLimit:  19997, // magic const for "disable"
		KWhPer100:            info.KWhPer100,
		AmountPerKWh:         info.AmountPerKWh,
		Efficiency:           info.Efficiency,
		BLETransmissionPower: info.BLETransmissionPower,
		PauseCharging:        wb.pauseCharging, // apply last value
		Current:              wb.current,       // apply last value
	}
}

// Status implements the api.Charger interface
func (wb *NRGKickBLE) Status() (api.ChargeStatus, error) {
	var res ble.Power
	if err := wb.read(ble.PowerService, &res); err != nil {
		return api.StatusNone, err
	}

	wb.log.TRACE.Printf("read power: %+v", res)

	switch res.CPSignal {
	case 3:
		return api.StatusB, nil
	case 2:
		return api.StatusC, nil
	case 4:
		return api.StatusA, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", res.CPSignal)
	}
}

// Enabled implements the api.Charger interface
func (wb *NRGKickBLE) Enabled() (bool, error) {
	var res ble.Info
	if err := wb.read(ble.InfoService, &res); err != nil {
		return false, err
	}

	wb.log.TRACE.Printf("read info: %+v", res)

	// workaround internal NRGkick state change after connecting
	// https://github.com/evcc-io/evcc/pull/274
	return !res.PauseCharging || res.ChargingActive, nil
}

// Enable implements the api.Charger interface
func (wb *NRGKickBLE) Enable(enable bool) error {
	var res ble.Info
	if err := wb.read(ble.InfoService, &res); err != nil {
		return err
	}

	// workaround internal NRGkick state change after connecting
	// https://github.com/evcc-io/evcc/pull/274
	if !enable && res.PauseCharging {
		wb.pauseCharging = false
		settings := wb.mergeSettings(res)

		wb.log.TRACE.Printf("write settings (workaround): %+v", settings)
		if err := wb.write(ble.SettingsService, &settings); err != nil {
			return err
		}
	}

	wb.pauseCharging = !enable // use cached value to work around API roundtrip delay
	settings := wb.mergeSettings(res)

	wb.log.TRACE.Printf("write settings: %+v", settings)

	return wb.write(ble.SettingsService, &settings)
}

// MaxCurrent implements the api.Charger interface
func (wb *NRGKickBLE) MaxCurrent(current int64) error {
	var res ble.Info
	if err := wb.read(ble.InfoService, &res); err != nil {
		return err
	}

	wb.current = int(current) // use cached value to work around API roundtrip delay
	settings := wb.mergeSettings(res)

	wb.log.TRACE.Printf("write settings: %+v", settings)

	return wb.write(ble.SettingsService, &settings)
}

var _ api.Meter = (*NRGKickBLE)(nil)

// CurrentPower implements the api.Meter interface
func (wb *NRGKickBLE) CurrentPower() (float64, error) {
	var res ble.Power
	if err := wb.read(ble.PowerService, &res); err != nil {
		return 0, err
	}

	wb.log.TRACE.Printf("read power: %+v", res)

	return float64(res.TotalPower) * 10, nil
}

var _ api.MeterEnergy = (*NRGKickBLE)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *NRGKickBLE) TotalEnergy() (float64, error) {
	var res ble.Energy
	if err := wb.read(ble.EnergyService, &res); err != nil {
		return 0, err
	}

	wb.log.TRACE.Printf("read energy: %+v", res)

	return float64(res.TotalEnergy) / 1000, nil
}

var _ api.PhaseCurrents = (*NRGKickBLE)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *NRGKickBLE) Currents() (float64, float64, float64, error) {
	var res ble.VoltageCurrent
	if err := wb.read(ble.VoltageCurrentService, &res); err != nil {
		return 0, 0, 0, err
	}

	wb.log.TRACE.Printf("read voltage/current: %+v", res)

	return float64(res.CurrentL1) / 100,
		float64(res.CurrentL2) / 100,
		float64(res.CurrentL3) / 100,
		nil
}

// ChargedEnergy implements the ChargeRater interface
// NOTE: apparently shows energy of a stopped charging session, hence substituted by TotalEnergy
// func (wb *NRGKickBLE) ChargedEnergy() (float64, error) {
// 	res := ble.Energy{}
// 	if err := wb.read(ble.EnergyService, &res); err != nil {
// 		return 0, err
// 	}
// 	wb.log.TRACE.Printf("energy: %+v", res)
// 	return float64(res.EnergyLastCharge) / 1000, nil
// }
