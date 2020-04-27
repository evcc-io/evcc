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
	log     *util.Logger
	uri     string
	power   float64
	updated time.Time
	recv    chan sma.Telegram
	mux     sync.Mutex
	once    sync.Once
}

// NewSMAFromConfig creates a SMA Meter from generic config
func NewSMAFromConfig(log *util.Logger, other map[string]interface{}) api.Meter {
	sm := struct {
		URI string
	}{}
	util.DecodeOther(log, other, &sm)

	return NewSMA(sm.URI)
}

// NewSMA creates a SMA Meter
func NewSMA(uri string) *SMA {
	log := util.NewLogger("sma ")

	sm := &SMA{
		log:  log,
		uri:  uri,
		recv: make(chan sma.Telegram),
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

// receive processes the channel message containing the multicast data
func (sm *SMA) receive() {
	for msg := range sm.recv {
		if msg.Values == nil {
			continue
		}

		sm.mux.Lock()

		if power, ok := msg.Values[sma.ObisExportPower]; ok {
			sm.power = -power
			sm.updated = time.Now()
		} else if power, ok := msg.Values[sma.ObisImportPower]; ok {
			sm.power = power
			sm.updated = time.Now()
		} else {
			sm.log.WARN.Println("missing obis for import/export power")
		}

		sm.mux.Unlock()
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
