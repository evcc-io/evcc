package meter

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider/sma"
	"github.com/evcc-io/evcc/util"
	"gitlab.com/bboehmke/sunny"
)

// SMA supporting SMA Home Manager 2.0, SMA Energy Meter 30 and SMA inverter
type SMA struct {
	uri    string
	scale  float64
	device *sma.Device
}

func init() {
	registry.Add("sma", NewSMAFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateSMA -b *SMA -r api.Meter -t "api.Battery,Soc,func() (float64, error)" -t "api.BatteryCapacity,Capacity,func() float64"

// NewSMAFromConfig creates an SMA meter from generic config
func NewSMAFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		capacity                 `mapstructure:",squash"`
		URI, Password, Interface string
		Serial                   uint32
		Scale                    float64 // power only
	}{
		Password: "0000",
		Scale:    1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSMA(cc.URI, cc.Password, cc.Interface, cc.Serial, cc.Scale, cc.capacity.Decorator())
}

// NewSMA creates an SMA meter
func NewSMA(uri, password, iface string, serial uint32, scale float64, capacity func() float64) (api.Meter, error) {
	sm := &SMA{
		uri:   uri,
		scale: scale,
	}

	discoverer, err := sma.GetDiscoverer(iface)
	if err != nil {
		return nil, fmt.Errorf("discoverer: %w", err)
	}

	switch {
	case uri != "":
		sm.device, err = discoverer.DeviceByIP(uri, password)
		if err != nil {
			return nil, err
		}

	case serial > 0:
		sm.device = discoverer.DeviceBySerial(serial, password)
		if sm.device == nil {
			return nil, fmt.Errorf("device not found: %d", serial)
		}

	default:
		return nil, errors.New("missing uri or serial")
	}

	// call UpdateValues first to check if we get an error
	if err := sm.device.UpdateValues(); err != nil {
		return nil, err
	}

	// start update loop manually to get values as fast as possible
	sm.device.StartUpdateLoop()

	// decorate api.Battery in case of inverter
	var soc func() (float64, error)
	if !sm.device.IsEnergyMeter() {
		vals, err := sm.device.Values()
		if err != nil {
			return nil, err
		}

		if _, ok := vals[sunny.BatteryCharge]; ok {
			soc = sm.soc
		}
	}

	return decorateSMA(sm, soc, capacity), nil
}

// CurrentPower implements the api.Meter interface
func (sm *SMA) CurrentPower() (float64, error) {
	values, err := sm.device.Values()
	return sm.scale * (sma.AsFloat(values[sunny.ActivePowerPlus]) - sma.AsFloat(values[sunny.ActivePowerMinus])), err
}

var _ api.MeterEnergy = (*SMA)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (sm *SMA) TotalEnergy() (float64, error) {
	values, err := sm.device.Values()
	return sma.AsFloat(values[sunny.ActiveEnergyPlus]) / 3600000, err
}

var _ api.MeterCurrent = (*SMA)(nil)

// Currents implements the api.MeterCurrent interface
func (sm *SMA) Currents() (float64, float64, float64, error) {
	values, err := sm.device.Values()

	var powers [3]float64
	for i, id := range []sunny.ValueID{sunny.ActivePowerMinusL1, sunny.ActivePowerMinusL2, sunny.ActivePowerMinusL3} {
		if p := sma.AsFloat(values[id]); p > 0 {
			powers[i] = -p
		}
	}

	var currents [3]float64
	for i, id := range []sunny.ValueID{sunny.CurrentL1, sunny.CurrentL2, sunny.CurrentL3} {
		currents[i] = util.SignFromPower(sma.AsFloat(values[id]), powers[i])
	}

	return currents[0], currents[1], currents[2], err
}

var _ api.MeterVoltage = (*SMA)(nil)

// Voltages implements the api.MeterVoltage interface
func (sm *SMA) Voltages() (float64, float64, float64, error) {
	values, err := sm.device.Values()

	var voltages [3]float64
	for i, id := range []sunny.ValueID{sunny.VoltageL1, sunny.VoltageL2, sunny.VoltageL3} {
		voltages[i] = sma.AsFloat(values[id])
	}

	return voltages[0], voltages[1], voltages[2], err
}

var _ api.MeterPower = (*SMA)(nil)

// Powers implements the api.MeterPower interface
func (sm *SMA) Powers() (float64, float64, float64, error) {
	values, err := sm.device.Values()

	var powers [3]float64
	for i, id := range []sunny.ValueID{sunny.ActivePowerPlusL1, sunny.ActivePowerPlusL2, sunny.ActivePowerPlusL3} {
		powers[i] = sma.AsFloat(values[id])
	}
	for i, id := range []sunny.ValueID{sunny.ActivePowerMinusL1, sunny.ActivePowerMinusL2, sunny.ActivePowerMinusL3} {
		powers[i] -= sma.AsFloat(values[id])
	}

	return powers[0], powers[1], powers[2], err
}

// soc implements the api.Battery interface
func (sm *SMA) soc() (float64, error) {
	values, err := sm.device.Values()
	return sma.AsFloat(values[sunny.BatteryCharge]), err
}

var _ api.Diagnosis = (*SMA)(nil)

// Diagnose implements the api.Diagnosis interface
func (sm *SMA) Diagnose() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintf(w, "  IP:\t%s\n", sm.device.Address())
	fmt.Fprintf(w, "  Serial:\t%d\n", sm.device.SerialNumber())
	fmt.Fprintf(w, "  EnergyMeter:\t%v\n", sm.device.IsEnergyMeter())
	fmt.Fprintln(w)

	if values, err := sm.device.Values(); err == nil {
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
