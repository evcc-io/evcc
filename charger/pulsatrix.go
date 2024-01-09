package charger

/*
MIT License

Copyright (c) 2023-2024 pulsatrix gmbh
Copyright (c) 2019-2024 andig

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	"nhooyr.io/websocket"
)

// pulsatrix charger implementation
type Pulsatrix struct {
	log     *util.Logger
	mu      sync.Mutex
	conn    *websocket.Conn
	uri     string
	enabled bool
	quit    chan struct{}
	data    *util.Monitor[pulsatrixData]
}

type pulsatrixData struct {
	VehicleStatus   string     `json:"vehicleStatus"`
	LastActivePower float64    `json:"lastActivePower"`
	PhaseVoltage    [3]float64 `json:"voltage"`
	PhaseAmperage   [3]float64 `json:"amperage"`
	AmperageLimit   float64    `json:"amperageLimit"`
	EnergyImported  float64    `json:"energyImported"`
}

func init() {
	registry.Add("pulsatrix", NewPulsatrixFromConfig)
}

// NewPulsatrixtFromConfig creates a pulsatrix charger from generic config
func NewPulsatrixFromConfig(other map[string]interface{}) (api.Charger, error) {
	var cc struct {
		Host string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewPulsatrix(cc.Host)
}

// NewPulsatrix creates pulsatrix charger
func NewPulsatrix(hostname string) (*Pulsatrix, error) {
	wb := Pulsatrix{
		log:  util.NewLogger("pulsatrix"),
		uri:  fmt.Sprintf("ws://%s/api/ws", hostname),
		data: util.NewMonitor[pulsatrixData](15 * time.Second),
	}

	if err := wb.connectWs(); err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	return &wb, nil
}

// ConnectWs connects to a pulsatrix SECC websocket
func (c *Pulsatrix) connectWs() error {
	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

	c.log.TRACE.Printf("connecting to %s", c.uri)
	conn, _, err := websocket.Dial(ctx, c.uri, nil)
	if err != nil {
		return err
	}

	c.conn = conn

	// ensure evcc and SECC are in sync
	if err := c.Enable(false); err != nil {
		c.log.ERROR.Println(err)
	}
	c.quit = make(chan struct{})
	go c.wsReader()
	go c.heartbeat()

	return nil
}

// ReconnectWs reconnects to a pulsatrix SECC websocket
func (c *Pulsatrix) reconnectWs() {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = time.Second
	bo.MaxInterval = 1 * time.Minute
	bo.MaxElapsedTime = 0 * time.Second // retry forever; default is 15 min
	if err := backoff.Retry(c.connectWs, bo); err != nil {
		c.log.ERROR.Println(err)
	}
}

// WsReader runs a loop that reads messages from the websocket
func (c *Pulsatrix) wsReader() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
		defer cancel()

		messageType, message, err := c.conn.Read(ctx)
		if err != nil {
			c.log.ERROR.Println("read message:", err)
			break
		} else {
			c.parseWsMessage(messageType, message)
		}
	}

	c.mu.Lock()
	c.conn.Close(websocket.StatusNormalClosure, "Reconnecting")
	c.conn = nil
	close(c.quit)
	c.mu.Unlock()

	c.reconnectWs()
}

// wsWrite writes a message to the websocket
func (c *Pulsatrix) write(message string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
		defer cancel()

		if err := c.conn.Write(ctx, websocket.MessageText, []byte(message)); err != nil {
			return err
		}
	}
	return nil
}

// ParseWsMessage parses a message from the websocket
func (c *Pulsatrix) parseWsMessage(messageType websocket.MessageType, message []byte) {
	if messageType == websocket.MessageText {
		b := bytes.ReplaceAll(message, []byte(":NaN"), []byte(":null"))
		var parsedMessage struct {
			Message json.RawMessage `json:"message"`
		}

		if err := json.Unmarshal(b, &parsedMessage); err != nil {
			c.log.ERROR.Println(err)
			return
		}

		val, _ := c.data.Get()
		if err := json.Unmarshal(parsedMessage.Message, &val); err != nil {
			c.log.ERROR.Println(err)
		} else {
			c.data.Set(val)
		}
	}
}

// Heartbeat sends a heartbeat to the pulsatrix SECC
func (c *Pulsatrix) heartbeat() {
	ticker := time.NewTicker(3 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.Enable(c.enabled); err != nil {
				c.log.ERROR.Println(err)
			}
		case <-c.quit:
			return
		}
	}
}

// Status implements the api.Charger interface
func (c *Pulsatrix) Status() (api.ChargeStatus, error) {
	res, err := c.data.Get()
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatusString(res.VehicleStatus)
}

// Enabled implements the api.Charger interface
func (c *Pulsatrix) Enabled() (bool, error) {
	return verifyEnabled(c, c.enabled)
}

// Enable implements the api.Charger interface
func (c *Pulsatrix) Enable(enable bool) error {
	err := c.write("setEnabled\n" + strconv.FormatBool(enable))
	if err == nil {
		c.enabled = enable
	}
	return err
}

// MaxCurrent implements the api.CurrentLimiter interface
func (c *Pulsatrix) MaxCurrent(current int64) error {
	return c.MaxCurrentMillis(float64(current))
}

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *Pulsatrix) MaxCurrentMillis(current float64) error {
	return c.write("setCurrentLimit\n" + strconv.FormatFloat(current, 'f', 10, 64))
}

var _ api.CurrentGetter = (*Pulsatrix)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (c *Pulsatrix) GetMaxCurrent() (float64, error) {
	res, err := c.data.Get()
	return res.AmperageLimit, err
}

// CurrentPower implements the api.Meter interface
func (c *Pulsatrix) CurrentPower() (float64, error) {
	res, err := c.data.Get()
	return res.LastActivePower, err
}

var _ api.MeterEnergy = (*Pulsatrix)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *Pulsatrix) TotalEnergy() (float64, error) {
	res, err := c.data.Get()
	return res.EnergyImported, err
}

// Phases1p3p implements the api.PhaseSwitcher interface
func (c *Pulsatrix) Phases1p3p(phases int) error {
	return c.write("set1p3p\n" + strconv.Itoa(phases))
}

var _ api.PhaseCurrents = (*Pulsatrix)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *Pulsatrix) Currents() (float64, float64, float64, error) {
	res, err := c.data.Get()
	if err != nil {
		return 0, 0, 0, err
	}

	return res.PhaseAmperage[0], res.PhaseAmperage[1], res.PhaseAmperage[2], nil
}

var _ api.PhaseVoltages = (*Pulsatrix)(nil)

// Voltages implements the api.PhaseVoltages interface
func (c *Pulsatrix) Voltages() (float64, float64, float64, error) {
	res, err := c.data.Get()
	if err != nil {
		return 0, 0, 0, err
	}

	return res.PhaseVoltage[0], res.PhaseVoltage[1], res.PhaseVoltage[2], nil
}
