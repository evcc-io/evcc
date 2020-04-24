package meter

import (
	"errors"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/meter/smameter"
)

const (
	udpTimeout = 10
)

// SmaMeter supporting SMA Home Manager 2.0 and SMA Energy Meter 30
type SmaMeter struct {
	uri        string
	power      float64
	lastUpdate time.Time
	recv       chan smameter.SmaTelegramData
}

// NewSmaMeterFromConfig creates a SMA Meter from generic config
func NewSmaMeterFromConfig(log *api.Logger, other map[string]interface{}) api.Meter {
	sm := struct {
		URI string
	}{}
	api.DecodeOther(log, other, &sm)

	return NewSmaMeter(sm.URI)
}

// NewSmaMeter creates a SMA Meter
func NewSmaMeter(uri string) *SmaMeter {
	log := api.NewLogger("smameter")

	sm := &SmaMeter{
		uri:  uri,
		recv: make(chan smameter.SmaTelegramData),
	}

	if smameter.Instance == nil {
		smameter.Instance = smameter.New(log, sm.uri)
	}

	smameter.Instance.Subscribe(uri, sm.recv)

	sm.lastUpdate = time.Now()
	go sm.receive()

	return sm
}

// receive processes the channel message containing the multicast data
func (sm *SmaMeter) receive() {
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
func (sm *SmaMeter) CurrentPower() (float64, error) {
	if time.Since(sm.lastUpdate) > udpTimeout*time.Second {
		return 0, errors.New("recv timeout")
	}

	return sm.power, nil
}
