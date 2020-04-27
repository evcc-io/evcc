package meter

import (
	"errors"
	"sync"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/meter/sma"
)

const (
	udpTimeout  = 10 * time.Second
	waitTimeout = 50 * time.Millisecond // interval when waiting for initial value
)

// SMA supporting SMA Home Manager 2.0 and SMA Energy Meter 30
type SMA struct {
	log        *api.Logger
	uri        string
	power      float64
	lastUpdate time.Time
	recv       chan sma.TelegramData
	mux        sync.Mutex
	once       sync.Once
}

// NewSMAFromConfig creates a SMA Meter from generic config
func NewSMAFromConfig(log *api.Logger, other map[string]interface{}) api.Meter {
	sm := struct {
		URI string
	}{}
	api.DecodeOther(log, other, &sm)

	return NewSMA(sm.URI)
}

// NewSMA creates a SMA Meter
func NewSMA(uri string) *SMA {
	log := api.NewLogger("sma ")

	sm := &SMA{
		log:  log,
		uri:  uri,
		recv: make(chan sma.TelegramData),
	}

	if sma.Instance == nil {
		sma.Instance = sma.New(log, sm.uri)
	}

	sma.Instance.Subscribe(uri, sm.recv)

	go sm.receive()

	return sm
}

// waitForInitialValue makes sure we don't start with an error
func (sm *SMA) waitForInitialValue() {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	if sm.lastUpdate.IsZero() {
		sm.log.TRACE.Print("waiting for initial value")

		// wait for initial update
		for sm.lastUpdate.IsZero() {
			sm.mux.Unlock()
			time.Sleep(waitTimeout)
			sm.mux.Lock()
		}
	}
}

// receive processes the channel message containing the multicast data
func (sm *SMA) receive() {
	for msg := range sm.recv {
		if msg.Data == nil {
			continue
		}

		var powerIn, powerOut float64
		var ok bool

		if powerIn, ok = msg.Data[sma.ObisImportPower]; !ok {
			continue
		}

		if powerOut, ok = msg.Data[sma.ObisExportPower]; !ok {
			continue
		}

		sm.mux.Lock()
		sm.lastUpdate = time.Now()
		if powerOut > 0 {
			sm.power = -powerOut
		} else {
			sm.power = powerIn
		}
		sm.mux.Unlock()
	}
}

// CurrentPower implements the Meter.CurrentPower interface
func (sm *SMA) CurrentPower() (float64, error) {
	sm.once.Do(sm.waitForInitialValue)
	sm.mux.Lock()
	defer sm.mux.Unlock()

	if time.Since(sm.lastUpdate) > udpTimeout {
		return 0, errors.New("recv timeout")
	}

	return sm.power, nil
}
