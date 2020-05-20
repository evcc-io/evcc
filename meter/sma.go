package meter

import (
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/meter/sma"
	"github.com/andig/evcc/util"
)

const udpTimeout = 10 * time.Second

// values bundles SMA readings
type values struct {
	power     float64
	energy    float64
	currentL1 float64
	currentL2 float64
	currentL3 float64
}

// SMA supporting SMA Home Manager 2.0 and SMA Energy Meter 30
type SMA struct {
	log     *util.Logger
	mux     *util.Waiter
	uri     string
	serial  string
	values  values
	powerO  sma.Obis
	energyO sma.Obis
	recv    chan sma.Telegram
}

// NewSMAFromConfig creates a SMA Meter from generic config
func NewSMAFromConfig(log *util.Logger, other map[string]interface{}) api.Meter {
	cc := struct {
		URI, Serial, Power, Energy string
	}{}
	util.DecodeOther(log, other, &cc)

	return NewSMA(cc.URI, cc.Serial, cc.Power, cc.Energy)
}

// NewSMA creates a SMA Meter
func NewSMA(uri, serial, power, energy string) api.Meter {
	log := util.NewLogger("sma")

	sm := &SMA{
		mux:     util.NewWaiter(udpTimeout, func() { log.TRACE.Println("wait for initial value") }),
		log:     log,
		uri:     uri,
		serial:  serial,
		powerO:  sma.Obis(power),
		energyO: sma.Obis(energy),
		recv:    make(chan sma.Telegram),
	}

	if sma.Instance == nil {
		sma.Instance = sma.New(log)
	}

	// we only need to subscribe to one of the two possible identifiers
	if uri != "" {
		sma.Instance.Subscribe(uri, sm.recv)
	} else if serial != "" {
		sma.Instance.Subscribe(serial, sm.recv)
	} else {
		log.FATAL.Fatalf("config: missing uri or serial")
	}

	go sm.receive()

	// decorate api.MeterEnergy
	if energy != "" {
		return &SMAEnergy{SMA: sm}
	}

	return sm
}

// update the actual meter data
func (sm *SMA) updateMeterValues(msg sma.Telegram) {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	if sm.powerO != "" {
		// use user-defined obis
		if power, ok := msg.Values[sm.powerO]; ok {
			sm.values.power = power
			sm.mux.Update()
		}
	} else {
		sm.values.power = msg.Values[sma.ImportPower] - msg.Values[sma.ExportPower]
		sm.mux.Update()
	}

	if sm.energyO != "" {
		if energy, ok := msg.Values[sm.energyO]; ok {
			sm.values.energy = energy
			sm.mux.Update()
		} else {
			sm.log.WARN.Println("missing obis for energy")
		}
	}

	if currentL1, ok := msg.Values[sma.CurrentL1]; ok {
		sm.values.currentL1 = currentL1
		sm.mux.Update()
	} else {
		sm.log.WARN.Println("missing obis for currentL1")
	}

	if currentL2, ok := msg.Values[sma.CurrentL2]; ok {
		sm.values.currentL2 = currentL2
		sm.mux.Update()
	} else {
		sm.log.WARN.Println("missing obis for currentL2")
	}

	if currentL3, ok := msg.Values[sma.CurrentL3]; ok {
		sm.values.currentL3 = currentL3
		sm.mux.Update()
	} else {
		sm.log.WARN.Println("missing obis for currentL3")
	}
}

// receive processes the channel message containing the multicast data
func (sm *SMA) receive() {
	for msg := range sm.recv {
		if msg.Values == nil {
			continue
		}

		sm.updateMeterValues(msg)
	}
}

func (sm *SMA) hasValue() (values, error) {
	elapsed := sm.mux.LockWithTimeout()
	defer sm.mux.Unlock()

	if elapsed > 0 {
		return values{}, fmt.Errorf("recv timeout: %v", elapsed.Truncate(time.Second))
	}

	return sm.values, nil
}

// CurrentPower implements the Meter.CurrentPower interface
func (sm *SMA) CurrentPower() (float64, error) {
	values, err := sm.hasValue()
	return values.power, err
}

// Currents implements the MeterCurrent interface
func (sm *SMA) Currents() (float64, float64, float64, error) {
	values, err := sm.hasValue()
	return values.currentL1, sm.values.currentL2, sm.values.currentL3, err
}

// SMAEnergy decorates SMA with api.MeterEnergy interface
type SMAEnergy struct {
	*SMA
}

// TotalEnergy implements the api.MeterEnergy interface
func (sm *SMAEnergy) TotalEnergy() (float64, error) {
	values, err := sm.hasValue()
	return values.energy, err
}
