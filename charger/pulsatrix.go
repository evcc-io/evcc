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
	"nhooyr.io/websocket"
)

// pulsatrix charger implementation
type PulsatrixCharger struct {
	conn        *websocket.Conn
	ctx         context.Context
	hostname    string
	log         *util.Logger
	tryRead     int
	enState     bool
	bo          *backoff.ExponentialBackOff
	signaledAmp float64
	mutex       sync.Mutex
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
	var cc = struct {
		Host string
	}{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	return NewPulsatrix(cc.Host)
}

// NewPulsatrix creates pulsatrix charger
func NewPulsatrix(hostname string) (*PulsatrixCharger, error) {
	wb := PulsatrixCharger{}
	wb.log = util.NewLogger("pulsatrix")
	wb.hostname = hostname
	wb.bo = backoff.NewExponentialBackOff()
	wb.bo.InitialInterval = 30 * time.Second
	wb.bo.MaxInterval = 5 * time.Minute
	return &wb, wb.connectWs(hostname)
}

// ConnectWs connects to a pulsatrix SECC websocket
func (c *PulsatrixCharger) connectWs(hostname string) error {
	uri := fmt.Sprintf("ws://%s/api/ws", hostname)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	c.ctx = ctx
	conn, _, err := websocket.Dial(ctx, uri, nil)

	if err != nil {
		if strErr := fmt.Sprintf("%s", err); strErr == "websocket: bad handshake" {
			c.log.ERROR.Println("bad handshake. Make sure the IP and port are correct")
		} else {
			c.log.ERROR.Println("error connecting to websocket:", err)
			c.reconnectWs()
		}
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			c.log.ERROR.Println("Recovered from panic:", r)
		}
	}()

	c.enState = false
	c.conn = conn
	c.handleError(c.Enable(false))
	go c.wsReader()
	go c.heartbeat()
	return nil
}

// Heartbeat sends a heartbeat to the pulsatrix SECC
func (c *PulsatrixCharger) heartbeat() {
	for {
		if c.enState {
			c.handleError(c.Enable(true))
			c.handleError(c.MaxCurrentMillis(c.signaledAmp))
		} else {
			c.handleError(c.Enable(false))
		}
		time.Sleep(3 * time.Minute)
	}
}

func (c *PulsatrixCharger) handleError(err error) {
	if err != nil {
		c.log.ERROR.Println("error:", err)
	}
}

// ReconnectWs reconnects to a pulsatrix SECC websocket
func (c *PulsatrixCharger) reconnectWs() {
	backoff.RetryNotify(func() error {
		c.conn.Close(websocket.StatusAbnormalClosure, "done")
		err := c.connectWs(c.hostname)
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
	var breaker bool
	go func() {
		for !breaker {
			messageType, message, err := c.conn.Read(context.Background())
			if err != nil {
				if c.tryRead < 3 {
					c.tryRead++
					time.Sleep(3 * time.Second)
				} else {
					c.log.ERROR.Println("error reading message: ", err)
					c.tryRead = 0
					c.reconnectWs()
					breaker = true
					break
				}
			} else {
				c.parseWsMessage(messageType, message)
			}
		}
	}()
}

// ParseWsMessage parses a message from the websocket
func (c *PulsatrixCharger) parseWsMessage(messageType websocket.MessageType, message []byte) {
	if messageType == websocket.MessageText {
		b := bytes.ReplaceAll(message, []byte(":NaN"), []byte(":null"))
		stadtIdx := bytes.IndexByte(b, '{')
		c.handleError(json.Unmarshal(b[stadtIdx:], c))
	}
}

// wsWrite writes a message to the websocket
func (c *PulsatrixCharger) wsWrite(message []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.conn == nil {
		return fmt.Errorf("reconnecting...")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := c.conn.Write(ctx, websocket.MessageText, message)

	return err
}

// Status implements the api.Charger interface
func (c *PulsatrixCharger) Status() (api.ChargeStatus, error) {
	return api.ChargeStatusString(c.VehicleStatus)
}

// Enabled implements the api.Charger interface
func (c *PulsatrixCharger) Enabled() (bool, error) {
	return verifyEnabled(c, c.enState)
}

// Enable implements the api.Charger interface
func (c *PulsatrixCharger) Enable(enable bool) error {
	c.enState = enable
	return c.wsWrite([]byte("setEnabled\n" + strconv.FormatBool(c.enState)))
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
	return c.wsWrite([]byte("setCurrentLimit\n" + res))
}

// GetMaxCurrent implements the api.CurrentGetter interface
func (c *PulsatrixCharger) GetMaxCurrent() (float64, error) {
	return float64(c.AllocatedAmperage), nil
}

// CurrentPower implements the api.Meter interface
func (c *PulsatrixCharger) CurrentPower() (float64, error) {
	return float64(c.LastActivePower), nil
}

// StartCharge implements the api.VehicleChargeController interface
func (c *PulsatrixCharger) StartCharge() error {
	return c.wsWrite([]byte("setCurrentLimit\n6"))
}

// StopCharge implements the api.VehicleChargeController interface
func (c *PulsatrixCharger) StopCharge() error {
	return c.wsWrite([]byte("setCurrentLimit\n0"))
}

// Currents implements the api.PhaseCurrents interface
func (c *PulsatrixCharger) Currents() (float64, float64, float64, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	currents := c.PhaseAmperage
	if len(currents) < 3 {
		return 0, 0, 0, fmt.Errorf("missing current data")
	}
	return currents[0], currents[1], currents[2], nil
}

// Voltages implements the api.PhaseVoltages interface
func (c *PulsatrixCharger) Voltages() (float64, float64, float64, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	voltages := c.PhaseVoltage
	if len(voltages) < 3 {
		return 0, 0, 0, fmt.Errorf("missing voltage data")
	}
	return voltages[0], voltages[1], voltages[2], nil
}

// Total Energy implements the api.MeterEnergy interface
func (c *PulsatrixCharger) TotalEnergy() (float64, error) {
	return float64(c.EnergyImported), nil
}
