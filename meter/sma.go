package meter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"text/tabwriter"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/imdario/mergo"
	"gitlab.com/bboehmke/sunny"
)

const udpTimeout = 10 * time.Second

// smaDiscoverer discovers SMA devices in background while providing already found devices
type smaDiscoverer struct {
	conn    *sunny.Connection
	devices map[uint32]*sunny.Device
	mux     sync.RWMutex
	done    uint32
}

// run discover and store found devices
func (d *smaDiscoverer) run() {
	devices := make(chan *sunny.Device)

	go func() {
		for device := range devices {
			d.mux.Lock()
			d.devices[device.SerialNumber()] = device
			d.mux.Unlock()
		}
	}()

	// discover devices and wait for results
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	d.conn.DiscoverDevices(ctx, devices, "")
	cancel()
	close(devices)

	// mark discover as done
	atomic.AddUint32(&d.done, 1)
}

func (d *smaDiscoverer) get(serial uint32) *sunny.Device {
	d.mux.RLock()
	defer d.mux.RUnlock()
	return d.devices[serial]
}

// deviceBySerial with the given serial number
func (d *smaDiscoverer) deviceBySerial(serial uint32) *sunny.Device {
	start := time.Now()
	for time.Since(start) < time.Second*3 {
		// discover done -> return immediately regardless of result
		if atomic.LoadUint32(&d.done) != 0 {
			return d.get(serial)
		}

		// device with serial found -> return
		if device := d.get(serial); device != nil {
			return device
		}

		time.Sleep(time.Millisecond * 10)
	}
	return d.get(serial)
}

// SMA supporting SMA Home Manager 2.0 and SMA Energy Meter 30
type SMA struct {
	log    *util.Logger
	mux    *util.Waiter
	uri    string
	iface  string
	values map[sunny.ValueID]interface{}
	scale  float64
	device *sunny.Device
}

func init() {
	registry.Add("sma", NewSMAFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateSMA -r api.Meter -b *SMA -t "api.Battery,SoC,func() (float64, error)"

// NewSMAFromConfig creates a SMA Meter from generic config
func NewSMAFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI, Password, Interface string
		Serial                   uint32
		Power, Energy            string
		Scale                    float64 // power only
	}{
		Password: "0000",
		Scale:    1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Power != "" || cc.Energy != "" {
		util.NewLogger("sma").WARN.Println("energy and power setting are deprecated and will be removed in a future release")
	}

	return NewSMA(cc.URI, cc.Password, cc.Interface, cc.Serial, cc.Scale)
}

// map of created discover instances
var discoverers = make(map[string]*smaDiscoverer)

// initialize sunny logger only once
var once sync.Once

// NewSMA creates a SMA Meter
func NewSMA(uri, password, iface string, serial uint32, scale float64) (api.Meter, error) {
	log := util.NewLogger("sma")
	once.Do(func() {
		sunny.Log = log.TRACE
	})

	sm := &SMA{
		mux:    util.NewWaiter(udpTimeout, func() { log.TRACE.Println("wait for initial value") }),
		log:    log,
		uri:    uri,
		iface:  iface,
		values: make(map[sunny.ValueID]interface{}),
		scale:  scale,
	}

	conn, err := sunny.NewConnection(iface)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}

	switch {
	case uri != "":
		sm.device, err = conn.NewDevice(uri, password)
		if err != nil {
			return nil, err
		}

	case serial > 0:
		discoverer, ok := discoverers[iface]
		if !ok {
			discoverer = &smaDiscoverer{
				conn:    conn,
				devices: make(map[uint32]*sunny.Device),
			}

			go discoverer.run()

			discoverers[iface] = discoverer
		}

		sm.device = discoverer.deviceBySerial(serial)
		if sm.device == nil {
			return nil, fmt.Errorf("device not found: %d", serial)
		}
		sm.device.SetPassword(password)

	default:
		return nil, errors.New("missing uri or serial")
	}

	// decorate api.Battery in case of inverter
	var soc func() (float64, error)
	if !sm.device.IsEnergyMeter() {
		vals, err := sm.device.GetValues()
		if err != nil {
			return nil, err
		}

		if _, ok := vals[sunny.BatteryCharge]; ok {
			soc = sm.soc
		}
	}

	go func() {
		for range time.NewTicker(time.Second).C {
			sm.updateValues()
		}
	}()

	return decorateSMA(sm, soc), nil
}

func (sm *SMA) updateValues() {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	values, err := sm.device.GetValues()
	if err == nil {
		err = mergo.Merge(&sm.values, values, mergo.WithOverride)
	}

	if err == nil {
		sm.mux.Update()
	} else {
		sm.log.ERROR.Println(err)
	}
}

func (sm *SMA) hasValue() (map[sunny.ValueID]interface{}, error) {
	elapsed := sm.mux.LockWithTimeout()
	defer sm.mux.Unlock()

	if elapsed > 0 {
		return nil, fmt.Errorf("update timeout: %v", elapsed.Truncate(time.Second))
	}

	return sm.values, nil
}

// CurrentPower implements the api.Meter interface
func (sm *SMA) CurrentPower() (float64, error) {
	values, err := sm.hasValue()
	return sm.scale * (sm.asFloat(values[sunny.ActivePowerPlus]) - sm.asFloat(values[sunny.ActivePowerMinus])), err
}

// TotalEnergy implements the api.MeterEnergy interface
func (sm *SMA) TotalEnergy() (float64, error) {
	values, err := sm.hasValue()
	return sm.asFloat(values[sunny.ActiveEnergyPlus]) / 3600000, err
}

// Currents implements the api.MeterCurrent interface
func (sm *SMA) Currents() (float64, float64, float64, error) {
	values, err := sm.hasValue()

	measurements := []sunny.ValueID{sunny.CurrentL1, sunny.CurrentL2, sunny.CurrentL3}
	var vals [3]float64
	for i := 0; i < 3; i++ {
		vals[i] = sm.asFloat(values[measurements[i]])
	}

	return vals[0], vals[1], vals[2], err
}

// soc implements the api.Battery interface
func (sm *SMA) soc() (float64, error) {
	values, err := sm.hasValue()
	return sm.asFloat(values[sunny.BatteryCharge]), err
}

// Diagnose implements the api.Diagnosis interface
func (sm *SMA) Diagnose() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintf(w, "  IP:\t%s\n", sm.device.Address())
	fmt.Fprintf(w, "  Serial:\t%d\n", sm.device.SerialNumber())
	fmt.Fprintf(w, "  EnergyMeter:\t%v\n", sm.device.IsEnergyMeter())
	fmt.Fprintln(w)

	if name, err := sm.device.GetDeviceName(); err == nil {
		fmt.Fprintf(w, "  Name:\t%s\n", name)
	}

	if devClass, err := sm.device.GetDeviceClass(); err == nil {
		fmt.Fprintf(w, "  Device Class:\t0x%X\n", devClass)
	}
	fmt.Fprintln(w)

	if values, err := sm.device.GetValues(); err == nil {
		ids := make([]sunny.ValueID, 0, len(values))
		for k := range values {
			ids = append(ids, k)
		}

		sort.Slice(ids, func(i, j int) bool {
			return ids[i].String() < ids[j].String()
		})

		for _, id := range ids {
			switch values[id].(type) {
			case float64:
				fmt.Fprintf(w, "  %s:\t%f %s\n", id.String(), values[id], sunny.GetValueInfo(id).Unit)
			default:
				fmt.Fprintf(w, "  %s:\t%v %s\n", id.String(), values[id], sunny.GetValueInfo(id).Unit)
			}
		}
	}
	w.Flush()
}

func (sm *SMA) asFloat(value interface{}) float64 {
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
	case nil:
		return 0
	default:
		sm.log.WARN.Printf("unknown value type: %T", value)
		return 0
	}
}
