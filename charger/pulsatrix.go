package charger

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
	"nhooyr.io/websocket"
)

// pulsatrix charger implementation
type PulsatrixCharger struct {
	conn        *websocket.Conn
	ctx         context.Context
	uri         string
	log         *util.Logger
	tryRead     int
	enState     bool
	bo          *backoff.ExponentialBackOff
	signaledAmp float64
	mutex       sync.Mutex
	updated     time.Time
	WebsocketData
}

type WebsocketData struct {
	VehicleStatus     string    `json:"vehicleStatus"`
	LastActivePower   float64   `json:"lastActivePower"`
	PhaseVoltage      []float64 `json:"voltage"`
	PhaseAmperage     []float64 `json:"amperage"`
	AllocatedAmperage float64   `json:"allocatedAmperage"`
	EnergyImported    float64   `json:"energyImported"`
}

func init() {
	registry.Add("pulsatrix", NewPulsatrixFromConfig)
}

// NewPulsatrixtFromConfig creates a pulsatrix charger from generic config
func NewPulsatrixFromConfig(other map[string]interface{}) (api.Charger, error) {
	var cc struct{ Host string }
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	return NewPulsatrix(cc.Host)
}

// NewPulsatrix creates pulsatrix charger
func NewPulsatrix(hostname string) (*PulsatrixCharger, error) {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 30 * time.Second
	bo.MaxInterval = 5 * time.Minute
	uri := fmt.Sprintf("ws://%s/api/ws", hostname)

	wb := PulsatrixCharger{
		log: util.NewLogger("pulsatrix"),
		uri: uri,
		bo:  bo,
	}
	return &wb, wb.connectWs()
}

// ConnectWs connects to a pulsatrix SECC websocket
func (c *PulsatrixCharger) connectWs() error {
	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()
	c.ctx = ctx
	c.log.INFO.Printf("connecting to %s", c.uri)
	conn, _, err := websocket.Dial(ctx, c.uri, nil)

	if err != nil {
		if strErr := fmt.Sprintf("%s", err); strErr == "websocket: bad handshake" {
			err = fmt.Errorf("bad handshake. Make sure the IP and port are correct")
		} else {
			err = fmt.Errorf("error connecting to websocket: %v", err)
		}
		return err
	}

	c.enState = false
	c.conn = conn
	c.handleError(c.Enable(false))
	go c.wsReader()
	go c.heartbeat()
	return nil
}

// Heartbeat sends a heartbeat to the pulsatrix SECC
func (c *PulsatrixCharger) heartbeat() {
	for range time.Tick(time.Minute) {
		if c.enState {
			c.handleError(c.Enable(true))
		} else {
			c.handleError(c.Enable(false))
		}
	}
}

func (c *PulsatrixCharger) handleError(err error) {
	if err != nil {
		c.log.ERROR.Println("error:", err)
	}
}

// ReconnectWs reconnects to a pulsatrix SECC websocket
func (c *PulsatrixCharger) reconnectWs() {
	c.conn = nil
	c.log.INFO.Printf("reconnecting to %s", c.uri)
	backoff.RetryNotify(func() error {
		err := c.connectWs()
		if err != nil {
			c.log.WARN.Printf("lost connection, trying to reconnect\n")
		}
		return err
	}, c.bo, func(err error, duration time.Duration) {
		c.log.WARN.Printf("trying to reconnect in %s...\n", duration)
	})
}

// WsReader runs a loop that reads messages from the websocket
func (c *PulsatrixCharger) wsReader() {
	go func() {
		for {
			messageType, message, err := c.conn.Read(context.Background())
			if err != nil {
				c.log.ERROR.Println("error reading message: ", err)
				if c.conn != nil {
					c.conn.Close(websocket.StatusGoingAway, "Reconnecting")
					c.reconnectWs()
				}
				return
			} else {
				c.parseWsMessage(messageType, message)
				c.updated = time.Now()
			}
		}
	}()
	fmt.Println("after go func")
}

// ParseWsMessage parses a message from the websocket
func (c *PulsatrixCharger) parseWsMessage(messageType websocket.MessageType, message []byte) {
	if messageType == websocket.MessageText {
		b := bytes.ReplaceAll(message, []byte(":NaN"), []byte(":null"))
		stadtIdx := bytes.IndexByte(b, '{')
		c.handleError(json.Unmarshal(b[stadtIdx:], c))
	}
}

func (c *PulsatrixCharger) invalid() bool {
	return c.updated.Before(time.Now().Add(-30 * time.Second))
}

// wsWrite writes a message to the websocket
func (c *PulsatrixCharger) wsWrite(message string) error {
	fmt.Println("Message: ", message)
	if c.conn != nil {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tmpMsg := []byte(message)
		err := c.conn.Write(ctx, websocket.MessageText, tmpMsg)
		if err != nil {
			return err
		}
	} else if c.invalid() && c.conn != nil {
		c.reconnectWs()
	} else {
		c.log.ERROR.Println("error: no connection")
	}
	return nil
}

// Status implements the api.Charger interface
func (c *PulsatrixCharger) Status() (api.ChargeStatus, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return api.ChargeStatusString(c.VehicleStatus)
}

// Enabled implements the api.Charger interface
func (c *PulsatrixCharger) Enabled() (bool, error) {
	return verifyEnabled(c, c.enState)
}

// Enable implements the api.Charger interface
func (c *PulsatrixCharger) Enable(enable bool) error {
	c.enState = enable
	return c.wsWrite("setEnabled\n" + strconv.FormatBool(c.enState))
}

// MaxCurrent implements the api.CurrentLimiter interface
func (c *PulsatrixCharger) MaxCurrent(current int64) error {
	return c.MaxCurrentMillis(float64(current))
}

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *PulsatrixCharger) MaxCurrentMillis(current float64) error {
	c.signaledAmp = current
	if !c.enState && current > 0 {
		c.handleError(c.Enable(true))
	}
	res := strconv.FormatFloat(current, 'f', 10, 64)
	return c.wsWrite("setCurrentLimit\n" + res)
}

// GetMaxCurrent implements the api.CurrentGetter interface
func (c *PulsatrixCharger) GetMaxCurrent() (float64, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.invalid() {
		return 0, fmt.Errorf("no data received")
	}
	return float64(c.AllocatedAmperage), nil

}

// CurrentPower implements the api.Meter interface
func (c *PulsatrixCharger) CurrentPower() (float64, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return float64(c.LastActivePower), nil
}

// Currents implements the api.PhaseCurrents interface
func (c *PulsatrixCharger) Currents() (float64, float64, float64, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	currents := c.PhaseAmperage
	if len(currents) < 3 || c.invalid() {
		return 0, 0, 0, fmt.Errorf("missing current data")
	}
	return currents[0], currents[1], currents[2], nil
}

// Voltages implements the api.PhaseVoltages interface
func (c *PulsatrixCharger) Voltages() (float64, float64, float64, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	voltages := c.PhaseVoltage
	if len(voltages) < 3 || c.invalid() {
		return 0, 0, 0, fmt.Errorf("missing voltage data")
	}
	return voltages[0], voltages[1], voltages[2], nil
}

// Total Energy implements the api.MeterEnergy interface
func (c *PulsatrixCharger) TotalEnergy() (float64, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return float64(c.EnergyImported), nil
}
