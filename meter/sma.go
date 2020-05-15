package meter

import (
	"errors"
	"sync"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/meter/sma"
	"github.com/andig/evcc/util"
)

const (
	udpTimeout  = 10 * time.Second
	waitTimeout = 50 * time.Millisecond // interval when waiting for initial value
)

// SMA supporting SMA Home Manager 2.0 and SMA Energy Meter 30
type SMA struct {
	log       *util.Logger
	uri       string
	serial    string
	power     float64
	energy    float64
	currentL1 float64
	currentL2 float64
	currentL3 float64
	powerO    sma.Obis
	energyO   sma.Obis
	updated   time.Time
	recv      chan sma.Telegram
	mux       sync.Mutex
	once      sync.Once
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
	log := util.NewLogger("sma ")

	sm := &SMA{
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

// waitForInitialValue makes sure we don't start with an error
func (sm *SMA) waitForInitialValue() {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	if sm.updated.IsZero() {
		sm.log.TRACE.Print("waiting for initial value")

		// wait for initial update
		for sm.updated.IsZero() {
			sm.mux.Unlock()
			time.Sleep(waitTimeout)
			sm.mux.Lock()
		}
	}
}

// update the actual meter data
func (sm *SMA) updateMeterValues(msg sma.Telegram) {
	sm.mux.Lock()

	if sm.powerO != "" {
		// use user-defined obis
		if power, ok := msg.Values[sm.powerO]; ok {
			sm.power = power
			sm.updated = time.Now()
		}
	} else {
		sm.power = msg.Values[sma.ImportPower] - msg.Values[sma.ExportPower]
		sm.updated = time.Now()
	}

	if sm.energyO != "" {
		if energy, ok := msg.Values[sm.energyO]; ok {
			sm.energy = energy
			sm.updated = time.Now()
		} else {
			sm.log.WARN.Println("missing obis for energy")
		}
	}

	if currentL1, ok := msg.Values[sma.CurrentL1]; ok {
		sm.currentL1 = currentL1
		sm.updated = time.Now()
	} else {
		sm.log.WARN.Println("missing obis for currentL1")
	}

	if currentL2, ok := msg.Values[sma.CurrentL2]; ok {
		sm.currentL2 = currentL2
		sm.updated = time.Now()
	} else {
		sm.log.WARN.Println("missing obis for currentL2")
	}

	if currentL3, ok := msg.Values[sma.CurrentL3]; ok {
		sm.currentL3 = currentL3
		sm.updated = time.Now()
	} else {
		sm.log.WARN.Println("missing obis for currentL3")
	}

	sm.mux.Unlock()
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

// CurrentPower implements the Meter.CurrentPower interface
func (sm *SMA) CurrentPower() (float64, error) {
	sm.once.Do(sm.waitForInitialValue)
	sm.mux.Lock()
	defer sm.mux.Unlock()

	if time.Since(sm.updated) > udpTimeout {
		return 0, errors.New("recv timeout")
	}

	return sm.power, nil
}

// Currents implements the MeterCurrent interface
func (sm *SMA) Currents() (float64, float64, float64, error) {
	sm.once.Do(sm.waitForInitialValue)
	sm.mux.Lock()
	defer sm.mux.Unlock()

	if time.Since(sm.updated) > udpTimeout {
		return 0, 0, 0, errors.New("recv timeout")
	}

	return sm.currentL1, sm.currentL2, sm.currentL3, nil
}

// SMAEnergy decorates SMA with api.MeterEnergy interface
type SMAEnergy struct {
	*SMA
}

// TotalEnergy implements the api.MeterEnergy interface
func (sm *SMAEnergy) TotalEnergy() (float64, error) {
	sm.once.Do(sm.waitForInitialValue)
	sm.mux.Lock()
	defer sm.mux.Unlock()

	if time.Since(sm.updated) > udpTimeout {
		return 0, errors.New("recv timeout")
	}

	return sm.energy, nil
}
