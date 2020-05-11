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
	"github.com/godbus/dbus"
	"github.com/lunixbochs/struc"
	"github.com/muka/go-bluetooth/bluez/profile/adapter"
	"github.com/muka/go-bluetooth/bluez/profile/agent"
	"github.com/muka/go-bluetooth/bluez/profile/device"
	"github.com/muka/go-bluetooth/hw"
)

// NRGKickBLE charger implementation
type NRGKickBLE struct {
	log        *util.Logger
	timer      *time.Timer
	adapter    *adapter.Adapter1
	agent      *agent.SimpleAgent
	dev        *device.Device1
	device     string
	macaddress string
	pin        int
}

// NewNRGKickBLEFromConfig creates a NRGKickBLE charger from generic config
func NewNRGKickBLEFromConfig(log *util.Logger, other map[string]interface{}) api.Charger {
	cc := struct{ Device, MacAddress, PIN string }{
		Device: "hci0",
	}
	util.DecodeOther(log, other, &cc)

	// decode PIN with leading zero
	pin, err := strconv.Atoi(cc.PIN)
	if err != nil {
		log.FATAL.Fatalf("config: invalid pin '%s'", cc.PIN)
	}

	return NewNRGKickBLE(cc.Device, cc.MacAddress, pin)
}

// NewNRGKickBLE creates NRGKickBLE charger
func NewNRGKickBLE(device, macaddress string, pin int) *NRGKickBLE {
	logger := util.NewLogger("nrgbt")
	logger.WARN.Println("-- experimental --")

	// set LE mode
	btmgmt := hw.NewBtMgmt(device)

	if len(os.Getenv("DOCKER")) > 0 {
		btmgmt.BinPath = "./bin/docker-btmgmt"
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
		logger.FATAL.Fatal(err)
	}

	adapt, err := adapter.NewAdapter1FromAdapterID(device)
	if err != nil {
		logger.FATAL.Fatal(err)
	}

	//Connect DBus System bus
	conn, err := dbus.SystemBus()
	if err != nil {
		logger.FATAL.Fatal(err)
	}

	// do not reuse agent0 from service
	agent.NextAgentPath()

	ag := agent.NewSimpleAgent()
	err = agent.ExposeAgent(conn, ag, agent.CapNoInputNoOutput, true)
	if err != nil {
		logger.FATAL.Fatal(err)
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

	return nrg
}

func (nrg *NRGKickBLE) connect() (*device.Device1, error) {
	dev, err := nrgble.FindDevice(nrg.adapter, nrg.macaddress)
	if err != nil {
		return nil, fmt.Errorf("findDevice: %s", err)
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

func (nrg *NRGKickBLE) read(service string) ([]byte, error) {
	nrg.waitTimer()
	defer nrg.setTimer()

	if nrg.dev == nil {
		dev, err := nrg.connect()
		if err != nil {
			return []byte{}, err
		}
		nrg.dev = dev
	}

	char, err := nrg.dev.GetCharByUUID(service)
	if err != nil {
		nrg.close()
		return []byte{}, err
	}

	b, err := char.ReadValue(map[string]interface{}{})
	if err != nil {
		nrg.close()
		return []byte{}, err
	}
	nrg.log.TRACE.Printf("read %s %0x", service, b)

	return b, nil
}

func (nrg *NRGKickBLE) write(service string, b []byte) error {
	nrg.waitTimer()
	defer nrg.setTimer()

	if nrg.dev == nil {
		dev, err := nrg.connect()
		if err != nil {
			return err
		}
		nrg.dev = dev
	}

	nrg.log.TRACE.Printf("write %s %0x", service, b)
	char, err := nrg.dev.GetCharByUUID(service)
	if err != nil {
		nrg.close()
		return err
	}

	if err := char.WriteValue(b, map[string]interface{}{}); err != nil {
		nrg.close()
		return err
	}

	return nil
}

func (nrg *NRGKickBLE) defaultSettings(info nrgble.Info) nrgble.Settings {
	return nrgble.Settings{
		PIN:                  nrg.pin,
		ChargingEnergyLimit:  19997, // magic const for "disable"
		KWhPer100:            info.KWhPer100,
		AmountPerKWh:         info.AmountPerKWh,
		Efficiency:           info.Efficiency,
		PauseCharging:        info.PauseCharging,
		BLETransmissionPower: info.BLETransmissionPower,
	}
}

// Status implements the Charger.Status interface
func (nrg *NRGKickBLE) Status() (api.ChargeStatus, error) {
	b, err := nrg.read(nrgble.PowerService)
	if err != nil {
		return api.StatusF, err
	}

	res := nrgble.Power{}
	if err := struc.Unpack(bytes.NewReader(b), &res); err != nil {
		return api.StatusF, err
	}
	nrg.log.TRACE.Printf("power: %+v", res)

	CPSignal := float64(res.CPSignal<<8)/100 + 0.4

	if CPSignal >= 1.3 && CPSignal <= 7.5 {
		return api.StatusC, nil
	}

	if CPSignal >= 1.3 && CPSignal <= 10.5 {
		return api.StatusB, nil
	}

	return api.StatusA, nil
}

// Enabled implements the Charger.Enabled interface
func (nrg *NRGKickBLE) Enabled() (bool, error) {
	b, err := nrg.read(nrgble.InfoService)
	if err != nil {
		nrg.log.TRACE.Println(err)
		return false, err
	}

	res := nrgble.Info{}
	if err := struc.Unpack(bytes.NewReader(b), &res); err != nil {
		return false, err
	}
	nrg.log.TRACE.Printf("info: %+v", res)

	return !res.PauseCharging, nil
}

// Enable implements the Charger.Enable interface
func (nrg *NRGKickBLE) Enable(enable bool) error {
	b, err := nrg.read(nrgble.InfoService)
	if err != nil {
		return err
	}

	res := nrgble.Info{}
	if err := struc.Unpack(bytes.NewReader(b), &res); err != nil {
		return err
	}
	nrg.log.TRACE.Printf("info: %+v", res)

	settings := nrg.defaultSettings(res)
	settings.PauseCharging = !enable

	var out bytes.Buffer
	if err := struc.Pack(&out, &settings); err != nil {
		return err
	}

	return nrg.write(nrgble.SettingsService, out.Bytes())
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (nrg *NRGKickBLE) MaxCurrent(current int64) error {
	b, err := nrg.read(nrgble.InfoService)
	if err != nil {
		return err
	}

	res := nrgble.Info{}
	if err := struc.Unpack(bytes.NewReader(b), &res); err != nil {
		return err
	}
	nrg.log.TRACE.Printf("info: %+v", res)

	settings := nrg.defaultSettings(res)
	settings.Current = int(current)

	var out bytes.Buffer
	if err := struc.Pack(&out, &settings); err != nil {
		return err
	}

	return nrg.write(nrgble.SettingsService, out.Bytes())
}

// CurrentPower implements the Meter interface.
func (nrg *NRGKickBLE) CurrentPower() (float64, error) {
	b, err := nrg.read(nrgble.PowerService)
	if err != nil {
		return 0, err
	}

	res := nrgble.Power{}
	if err := struc.Unpack(bytes.NewReader(b), &res); err != nil {
		return 0, err
	}
	nrg.log.TRACE.Printf("power: %+v", res)

	return float64(res.TotalPower) * 10, nil
}

// TotalEnergy implements the MeterEnergy interface.
func (nrg *NRGKickBLE) TotalEnergy() (float64, error) {
	b, err := nrg.read(nrgble.EnergyService)
	if err != nil {
		return 0, err
	}

	res := nrgble.Energy{}
	if err := struc.Unpack(bytes.NewReader(b), &res); err != nil {
		return 0, err
	}
	nrg.log.TRACE.Printf("energy: %+v", res)

	return float64(res.TotalEnergy) / 1000, nil
}

// Currents implements the MeterCurrent interface.
func (nrg *NRGKickBLE) Currents() (float64, float64, float64, error) {
	b, err := nrg.read(nrgble.VoltageCurrentService)
	if err != nil {
		return 0, 0, 0, err
	}

	res := nrgble.VoltageCurrent{}
	if err := struc.Unpack(bytes.NewReader(b), &res); err != nil {
		return 0, 0, 0, err
	}
	nrg.log.TRACE.Printf("voltage/current: %+v", res)

	return float64(res.CurrentL1) / 100,
		float64(res.CurrentL2) / 100,
		float64(res.CurrentL3) / 100,
		nil
}

// ChargedEnergy implements the ChargeRater interface.
// NOTE: apparently shows energy of a stopped charging session, hence substituted by TotalEnergy
// func (nrg *NRGKickBLE) ChargedEnergy() (float64, error) {
// 	nrg.log.TRACE.Print("read EnergyService")
// 	b, err := nrg.read(nrgble.EnergyService)
// 	if err != nil {
// 		return 0, err
// 	}

// 	res := nrgble.Energy{}
// 	if err := struc.Unpack(bytes.NewReader(b), &res); err != nil {
// 		return 0, err
// 	}

// 	nrg.log.TRACE.Printf("energy: %+v", res)

// 	return float64(res.EnergyLastCharge) / 1000, nil
// }
