package charger

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/charger/nrgble"
	"github.com/andig/evcc/util"
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
	log           *util.Logger
	timer         *time.Timer
	adapter       *adapter.Adapter1
	agent         *agent.SimpleAgent
	dev           *device.Device1
	device        string
	macaddress    string
	pin           int
	pauseCharging bool
	current       int
}

func init() {
	registry.Add("nrgkick-bluetooth", NewNRGKickBLEFromConfig)
}

// NewNRGKickBLEFromConfig creates a NRGKickBLE charger from generic config
func NewNRGKickBLEFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Device, MacAddress, PIN string `validate:"required"`
	}{
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

	return NewNRGKickBLE(cc.Device, cc.MacAddress, pin)
}

// NewNRGKickBLE creates NRGKickBLE charger
func NewNRGKickBLE(device, macaddress string, pin int) (*NRGKickBLE, error) {
	logger := util.NewLogger("nrg-bt")

	// set LE mode
	btmgmt := hw.NewBtMgmt(device)

	if len(os.Getenv("DOCKER")) > 0 {
		btmgmt.BinPath = "./docker-btmgmt"
	}

	err := btmgmt.SetPowered(false)
	if err == nil {
		err = btmgmt.SetLe(true)
		if err == nil {
			err = btmgmt.SetBredr(false)
			if err == nil {
				err = btmgmt.SetPowered(true)
			}
		}
	}

	if err != nil {
		return nil, err
	}

	adapt, err := adapter.NewAdapter1FromAdapterID(device)
	if err != nil {
		return nil, err
	}

	//Connect DBus System bus
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, err
	}

	// do not reuse agent0 from service
	agent.NextAgentPath()

	ag := agent.NewSimpleAgent()
	err = agent.ExposeAgent(conn, ag, agent.CapNoInputNoOutput, true)
	if err != nil {
		return nil, err
	}

	nrg := &NRGKickBLE{
		log:        logger,
		timer:      time.NewTimer(1),
		device:     device,
		macaddress: macaddress,
		pin:        pin,
		adapter:    adapt,
		agent:      ag,
	}

	return nrg, nil
}

func (nrg *NRGKickBLE) connect() (*device.Device1, error) {
	dev, err := nrgble.FindDevice(nrg.adapter, nrg.macaddress, nrgTimeout)
	if err != nil {
		return nil, fmt.Errorf("find device: %s", err)
	}

	err = nrgble.Connect(dev, nrg.agent, nrg.device)
	if err != nil {
		return nil, err
	}

	return dev, nil
}

func (nrg *NRGKickBLE) close() {
	if nrg.dev != nil {
		nrg.dev.Close()
		nrg.dev = nil
	}
}

func (nrg *NRGKickBLE) waitTimer() {
	<-nrg.timer.C
}

func (nrg *NRGKickBLE) setTimer() {
	nrg.timer.Stop() // can be stopped without reading as channel is always drained
	nrg.timer.Reset(2 * time.Second)
}

func (nrg *NRGKickBLE) read(service string, res interface{}) error {
	nrg.waitTimer()
	defer nrg.setTimer()

	if nrg.dev == nil {
		dev, err := nrg.connect()
		if err != nil {
			return err
		}
		nrg.dev = dev
	}

	char, err := nrg.dev.GetCharByUUID(service)
	if err != nil {
		nrg.close()
		return err
	}

	b, err := char.ReadValue(map[string]interface{}{})
	if err != nil {
		nrg.close()
		return err
	}
	nrg.log.TRACE.Printf("read %s %0x", service, b)

	return struc.Unpack(bytes.NewReader(b), res)
}

func (nrg *NRGKickBLE) write(service string, val interface{}) error {
	var out bytes.Buffer
	if err := struc.Pack(&out, val); err != nil {
		return err
	}
	nrg.log.TRACE.Printf("write %s %0x", service, out.Bytes())

	nrg.waitTimer()
	defer nrg.setTimer()

	if nrg.dev == nil {
		dev, err := nrg.connect()
		if err != nil {
			return err
		}
		nrg.dev = dev
	}

	char, err := nrg.dev.GetCharByUUID(service)
	if err != nil {
		nrg.close()
		return err
	}

	if err := char.WriteValue(out.Bytes(), map[string]interface{}{}); err != nil {
		nrg.close()
		return err
	}

	return nil
}

func (nrg *NRGKickBLE) mergeSettings(info nrgble.Info) nrgble.Settings {
	return nrgble.Settings{
		PIN:                  nrg.pin,
		ChargingEnergyLimit:  19997, // magic const for "disable"
		KWhPer100:            info.KWhPer100,
		AmountPerKWh:         info.AmountPerKWh,
		Efficiency:           info.Efficiency,
		BLETransmissionPower: info.BLETransmissionPower,
		PauseCharging:        nrg.pauseCharging, // apply last value
		Current:              nrg.current,       // apply last value
	}
}

// Status implements the Charger.Status interface
func (nrg *NRGKickBLE) Status() (api.ChargeStatus, error) {
	res := nrgble.Power{}
	if err := nrg.read(nrgble.PowerService, &res); err != nil {
		return api.StatusF, err
	}

	nrg.log.TRACE.Printf("read power: %+v", res)

	switch res.CPSignal {
	case 3:
		return api.StatusB, nil
	case 2:
		return api.StatusC, nil
	case 4:
		return api.StatusA, nil
	}

	return api.StatusA, fmt.Errorf("unexpected cp signal: %d", res.CPSignal)
}

// Enabled implements the Charger.Enabled interface
func (nrg *NRGKickBLE) Enabled() (bool, error) {
	res := nrgble.Info{}
	if err := nrg.read(nrgble.InfoService, &res); err != nil {
		nrg.log.TRACE.Println(err)
		return false, err
	}

	nrg.log.TRACE.Printf("read info: %+v", res)

	// workaround internal NRGkick state change after connecting
	// https://github.com/andig/evcc/pull/274
	return !res.PauseCharging || res.ChargingActive, nil
}

// Enable implements the Charger.Enable interface
func (nrg *NRGKickBLE) Enable(enable bool) error {
	res := nrgble.Info{}
	if err := nrg.read(nrgble.InfoService, &res); err != nil {
		return err
	}

	// workaround internal NRGkick state change after connecting
	// https://github.com/andig/evcc/pull/274
	if !enable && res.PauseCharging {
		nrg.pauseCharging = false
		settings := nrg.mergeSettings(res)

		nrg.log.TRACE.Printf("write settings (workaround): %+v", settings)
		if err := nrg.write(nrgble.SettingsService, &settings); err != nil {
			return err
		}
	}

	nrg.pauseCharging = !enable // use cached value to work around API roundtrip delay
	settings := nrg.mergeSettings(res)

	nrg.log.TRACE.Printf("write settings: %+v", settings)

	return nrg.write(nrgble.SettingsService, &settings)
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (nrg *NRGKickBLE) MaxCurrent(current int64) error {
	res := nrgble.Info{}
	if err := nrg.read(nrgble.InfoService, &res); err != nil {
		return err
	}

	nrg.current = int(current) // use cached value to work around API roundtrip delay
	settings := nrg.mergeSettings(res)

	nrg.log.TRACE.Printf("write settings: %+v", settings)

	return nrg.write(nrgble.SettingsService, &settings)
}

// CurrentPower implements the Meter interface.
func (nrg *NRGKickBLE) CurrentPower() (float64, error) {
	res := nrgble.Power{}
	if err := nrg.read(nrgble.PowerService, &res); err != nil {
		return 0, err
	}

	nrg.log.TRACE.Printf("read power: %+v", res)

	return float64(res.TotalPower) * 10, nil
}

// TotalEnergy implements the MeterEnergy interface.
func (nrg *NRGKickBLE) TotalEnergy() (float64, error) {
	res := nrgble.Energy{}
	if err := nrg.read(nrgble.EnergyService, &res); err != nil {
		return 0, err
	}

	nrg.log.TRACE.Printf("read energy: %+v", res)

	return float64(res.TotalEnergy) / 1000, nil
}

// Currents implements the MeterCurrent interface.
func (nrg *NRGKickBLE) Currents() (float64, float64, float64, error) {
	res := nrgble.VoltageCurrent{}
	if err := nrg.read(nrgble.VoltageCurrentService, &res); err != nil {
		return 0, 0, 0, err
	}

	nrg.log.TRACE.Printf("read voltage/current: %+v", res)

	return float64(res.CurrentL1) / 100,
		float64(res.CurrentL2) / 100,
		float64(res.CurrentL3) / 100,
		nil
}

// ChargedEnergy implements the ChargeRater interface.
// NOTE: apparently shows energy of a stopped charging session, hence substituted by TotalEnergy
// func (nrg *NRGKickBLE) ChargedEnergy() (float64, error) {
// 	res := nrgble.Energy{}
// 	if err := nrg.read(nrgble.EnergyService, &res); err != nil {
// 		return 0, err
// 	}
// 	nrg.log.TRACE.Printf("energy: %+v", res)
// 	return float64(res.EnergyLastCharge) / 1000, nil
// }
