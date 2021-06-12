package meter

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"gitlab.com/bboehmke/sunny"
)

const udpTimeout = 10 * time.Second

// SMADiscoverHelper discovers SMA devices in background while providing already found devices
type SMADiscoverHelper struct {
	conn         *sunny.Connection
	devices      map[string]*sunny.Device
	devicesMutex sync.RWMutex
	done         uint32
}

// NewSMADiscoverHelper creates discovery helper for given connection (normally one per interface)
func NewSMADiscoverHelper(conn *sunny.Connection) *SMADiscoverHelper {
	discover := SMADiscoverHelper{
		conn:    conn,
		devices: make(map[string]*sunny.Device),
	}

	go discover.run()

	return &discover
}

// run discover and store found devices
func (d *SMADiscoverHelper) run() {
	devices := make(chan *sunny.Device, 10)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for device := range devices {
			d.devicesMutex.Lock()
			d.devices[strconv.FormatInt(int64(device.SerialNumber()), 10)] = device
			d.devicesMutex.Unlock()
		}
		wg.Done()
	}()

	// discover devices and wait for results
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	d.conn.DiscoverDevices(ctx, devices, "")
	cancel()
	close(devices)
	wg.Wait()

	// mark discover as done
	atomic.AddUint32(&d.done, 1)
}

func (d *SMADiscoverHelper) get(serial string) *sunny.Device {
	d.devicesMutex.RLock()
	defer d.devicesMutex.RUnlock()
	return d.devices[serial]
}

// GetDevice with the given serial number
func (d *SMADiscoverHelper) GetDevice(serial string) *sunny.Device {
	start := time.Now()
	for time.Since(start) < time.Second*3 {
		// discover done -> return immediately regardless of result
		if atomic.LoadUint32(&d.done) != 0 {
			return d.get(serial)

			// device with serial found -> return
		} else if device := d.get(serial); device != nil {
			return device
		}

		time.Sleep(time.Millisecond * 10)
	}
	return d.get(serial)
}

// values bundles SMA readings
type values struct {
	power     float64
	energy    float64
	currentL1 float64
	currentL2 float64
	currentL3 float64
	soc       float64
}

// SMA supporting SMA Home Manager 2.0 and SMA Energy Meter 30
type SMA struct {
	log    *util.Logger
	mux    *util.Waiter
	uri    string
	serial string
	iface  string
	values values
	scale  float64

	device       *sunny.Device
	updateTicker *time.Ticker
}

func init() {
	registry.Add("sma", NewSMAFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateSMA -r api.Meter -b *SMA -t "api.Battery,SoC,func() (float64, error)"

// NewSMAFromConfig creates a SMA Meter from generic config
func NewSMAFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI, Password, Serial, Interface, Power, Energy string
		Scale                                           float64
	}{
		Password: "0000",
		Scale:    1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSMA(cc.URI, cc.Password, cc.Serial, cc.Interface, cc.Power, cc.Energy, cc.Scale)
}

// map of created discover instances
var discovers = make(map[string]*SMADiscoverHelper)

// NewSMA creates a SMA Meter
func NewSMA(uri, password, serial, iface, power, energy string, scale float64) (api.Meter, error) {
	log := util.NewLogger("sma")
	sunny.Log = log.TRACE

	// print warnings for unused config
	if power != "" {
		log.WARN.Println("SMA power not supported -> ignoring")
	}
	if energy != "" {
		log.WARN.Println("SMA energy not supported -> ignoring")
	}

	sm := &SMA{
		mux:          util.NewWaiter(udpTimeout, func() { log.TRACE.Println("wait for initial value") }),
		log:          log,
		uri:          uri,
		serial:       serial,
		iface:        iface,
		updateTicker: time.NewTicker(time.Second),
		scale:        scale,
	}

	conn, err := sunny.NewConnection(iface)
	if err != nil {
		return nil, fmt.Errorf("failed to get SMA connection: %w", err)
	}

	if uri != "" {
		sm.device, err = conn.NewDevice(uri, password)
		if err != nil {
			return nil, err
		}
	} else if serial != "" {
		if _, ok := discovers[iface]; !ok {
			discovers[iface] = NewSMADiscoverHelper(conn)
		}

		sm.device = discovers[iface].GetDevice(serial)
		if sm.device == nil {
			return nil, fmt.Errorf("failed to find device with serial: %s", serial)
		}
		sm.device.SetPassword(password)
	} else {
		return nil, errors.New("missing uri or serial")
	}

	// decorate api.Battery
	var soc func() (float64, error)
	if !sm.device.IsEnergyMeter() { // only for inverters possible
		vals, err := sm.device.GetValues()
		if err != nil {
			return nil, err
		}

		if _, ok := vals["battery_charge"]; ok {
			soc = sm.soc
		}
	}

	go func() {
		for range sm.updateTicker.C {
			sm.updateValues()
		}
	}()

	return decorateSMA(sm, soc), nil
}

func (sm *SMA) updateValues() {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	vals, err := sm.device.GetValues()
	if err != nil {
		sm.log.ERROR.Printf("failed to get values: %v", err)
		return
	}

	if sm.device.IsEnergyMeter() {
		powerP, ok1 := vals["active_power_plus"]
		powerM, ok2 := vals["active_power_minus"]
		if ok1 && ok2 {
			sm.values.power = sm.scale * (sm.convertValue(powerP) - sm.convertValue(powerM))
			sm.mux.Update()
		} else {
			sm.log.ERROR.Println("missing value for power")
		}

		if currentL1, ok := vals["l1_current"]; ok {
			sm.values.currentL1 = sm.convertValue(currentL1)
			sm.mux.Update()
		} else {
			sm.log.ERROR.Println("missing value for currentL1")
		}

		if currentL2, ok := vals["l2_current"]; ok {
			sm.values.currentL2 = sm.convertValue(currentL2)
			sm.mux.Update()
		} else {
			sm.log.ERROR.Println("missing value for currentL2")
		}

		if currentL3, ok := vals["l3_current"]; ok {
			sm.values.currentL3 = sm.convertValue(currentL3)
			sm.mux.Update()
		} else {
			sm.log.ERROR.Println("missing value for currentL3")
		}

		if energyTotal, ok := vals["active_energy_plus"]; ok {
			sm.values.energy = sm.convertValue(energyTotal) / 3600000
			sm.mux.Update()
		}

	} else {
		if power, ok := vals["power_ac_total"]; ok {
			sm.values.power = sm.convertValue(power)
			sm.mux.Update()
		} else {
			sm.log.DEBUG.Println("missing value for power -> set to 0")
			sm.values.power = 0
			sm.mux.Update()
		}

		if currentL1, ok := vals["current_ac1"]; ok {
			sm.values.currentL1 = sm.convertValue(currentL1) / 1000
			sm.mux.Update()
		}

		if currentL2, ok := vals["current_ac2"]; ok {
			sm.values.currentL2 = sm.convertValue(currentL2) / 1000
			sm.mux.Update()
		}

		if currentL3, ok := vals["current_ac3"]; ok {
			sm.values.currentL3 = sm.convertValue(currentL3) / 1000
			sm.mux.Update()
		}

		if soc, ok := vals["battery_charge"]; ok {
			sm.values.soc = sm.convertValue(soc)
			sm.mux.Update()
		}

		if energyTotal, ok := vals["energy_total"]; ok {
			sm.values.energy = sm.convertValue(energyTotal) / 1000
			sm.mux.Update()
		}
	}
}

func (sm *SMA) hasValue() (values, error) {
	elapsed := sm.mux.LockWithTimeout()
	defer sm.mux.Unlock()

	if elapsed > 0 {
		return values{}, fmt.Errorf("update timeout: %v", elapsed.Truncate(time.Second))
	}

	return sm.values, nil
}

// CurrentPower implements the api.Meter interface
func (sm *SMA) CurrentPower() (float64, error) {
	values, err := sm.hasValue()
	return values.power, err
}

// Currents implements the api.MeterCurrent interface
func (sm *SMA) Currents() (float64, float64, float64, error) {
	values, err := sm.hasValue()
	return values.currentL1, values.currentL2, values.currentL3, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (sm *SMA) TotalEnergy() (float64, error) {
	values, err := sm.hasValue()
	return values.energy, err
}

// soc implements the api.Battery interface
func (sm *SMA) soc() (float64, error) {
	values, err := sm.hasValue()
	return values.soc, err
}

// Diagnose implements the api.Diagnosis interface
func (sm *SMA) Diagnose() {
	fmt.Printf("  IP:             %s\n", sm.device.Address())
	fmt.Printf("  Serial:         %d\n", sm.device.SerialNumber())
	fmt.Printf("  Is EnergyMeter: %v\n", sm.device.IsEnergyMeter())
	fmt.Printf("\n")
	name, err := sm.device.GetDeviceName()
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
	} else {
		fmt.Printf("  Name: %s\n", name)
	}
	devClass, err := sm.device.GetDeviceClass()
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
	} else {
		fmt.Printf("  Device Class: 0x%X\n", devClass)
	}
	fmt.Printf("\n")
	values, err := sm.device.GetValues()
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
	} else {
		keys := make([]string, 0, len(values))
		keyLength := 0
		for k := range values {
			keys = append(keys, k)
			if len(k) > keyLength {
				keyLength = len(k)
			}
		}
		sort.Strings(keys)

		for _, k := range keys {
			fmt.Printf("  %s:%s %v %s\n", k, strings.Repeat(" ", keyLength-len(k)), values[k], sm.device.GetValueInfo(k).Unit)
		}
	}
}

func (sm *SMA) convertValue(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	default:
		sm.log.WARN.Printf("unknown value type: %s", reflect.TypeOf(value).Name())
		return 0
	}
}
