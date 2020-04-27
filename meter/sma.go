package meter

import (
	"errors"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/meter/sma"
)

const (
	udpTimeout = 10 * time.Second
)

// SMA supporting SMA Home Manager 2.0 and SMA Energy Meter 30
type SMA struct {
	uri        string
	power      float64
	lastUpdate time.Time
	recv       chan sma.TelegramData
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
		uri:  uri,
		recv: make(chan sma.TelegramData),
	}

	if sma.Instance == nil {
		sma.Instance = sma.New(log, sm.uri)
	}

	sma.Instance.Subscribe(uri, sm.recv)

	sm.lastUpdate = time.Now()
	go sm.receive()

	return sm
}

// receive processes the channel message containing the multicast data
func (sm *SMA) receive() {
	for {
		msg := <-sm.recv
		if msg.Data == nil {
			return
		}

		var powerIn, powerOut float64
		foundValues := false
		for _, element := range msg.Data {
			if element.ObisCode == "1:1.4.0" {
				powerIn = element.Value
				foundValues = true
			}
			if element.ObisCode == "1:2.4.0" {
				powerOut = element.Value
				foundValues = true
			}
		}
		if foundValues {
			sm.lastUpdate = time.Now()
			if powerOut > 0 {
				sm.power = powerOut * -1
			} else {
				sm.power = powerIn
			}
		}
	}
}

// CurrentPower implements the Meter.CurrentPower interface
func (sm *SMA) CurrentPower() (float64, error) {
	if time.Since(sm.lastUpdate) > udpTimeout {
		return 0, errors.New("recv timeout")
	}

	return sm.power, nil
}
