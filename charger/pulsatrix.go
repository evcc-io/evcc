package charger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	request.Timeout = 15 * time.Second
	wb := PulsatrixCharger{
		log:     util.NewLogger("pulsatrix"),
		uri:     fmt.Sprintf("ws://%s/api/ws", hostname),
		bo:      bo,
		updated: time.Now(),
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
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("Make sure the IP is correct and the SECC is connected to the network")
		} else {
			err = fmt.Errorf("error connecting to websocket: %v", err)
		}
		return err
	}

	//ensure evcc and SECC are in sync
	c.handleError(c.Enable(false))

	c.conn = conn
	go c.wsReader()
	go c.heartbeat()
	return nil
}

// ReconnectWs reconnects to a pulsatrix SECC websocket
func (c *PulsatrixCharger) reconnectWs() {
	notify := func(err error, time time.Duration) {
		c.log.WARN.Printf("trying to reconnect in %v...\n", time)
	}
	c.handleError(backoff.RetryNotify(c.connectWs, c.bo, notify))
}

// WsReader runs a loop that reads messages from the websocket
func (c *PulsatrixCharger) wsReader() {
	for c.valid() {
		ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
		defer cancel()
		messageType, message, err := c.conn.Read(ctx)
		if err != nil {
			c.log.ERROR.Println("error reading message:", err)
			break
		} else {
			c.parseWsMessage(messageType, message)
			c.updated = time.Now()
		}
	}
	fmt.Println("wsReader stopped")
	c.mutex.Lock()
	if c.conn != nil {
		c.conn.Close(websocket.StatusNormalClosure, "Reconnecting")
		c.conn = nil
	}
	c.mutex.Unlock()
	c.reconnectWs()
}

// wsWrite writes a message to the websocket
func (c *PulsatrixCharger) wsWriter(message string) error {
	if c.valid() && c.conn != nil {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
		defer cancel()
		err := c.conn.Write(ctx, websocket.MessageText, []byte(message))
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Waiting for reconnect")
	}
	return nil
}

// ParseWsMessage parses a message from the websocket
func (c *PulsatrixCharger) parseWsMessage(messageType websocket.MessageType, message []byte) {
	if messageType == websocket.MessageText {
		b := bytes.ReplaceAll(message, []byte(":NaN"), []byte(":null"))
		idx := bytes.IndexByte(b, '{')
		c.handleError(json.Unmarshal(b[idx:], c))
	}
}

// Heartbeat sends a heartbeat to the pulsatrix SECC
func (c *PulsatrixCharger) heartbeat() {
	for range time.Tick(3 * time.Minute) {
		c.handleError(c.Enable(c.enState))
	}
}

func (c *PulsatrixCharger) valid() bool {
	return time.Since(c.updated) < 30*time.Second
}

func (c *PulsatrixCharger) handleError(err error) {
	if err != nil {
		c.log.ERROR.Println(err)
	}
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
	return c.wsWriter("setEnabled\n" + strconv.FormatBool(c.enState))
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
	return c.wsWriter("setCurrentLimit\n" + strconv.FormatFloat(current, 'f', 10, 64))
}

// GetMaxCurrent implements the api.CurrentGetter interface
func (c *PulsatrixCharger) GetMaxCurrent() (float64, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if !c.valid() {
		c.handleError(c.Enable(false))
		return 0, fmt.Errorf(api.ErrOutdated.Error())
	}
	return float64(c.AllocatedAmperage), nil

}

// CurrentPower implements the api.Meter interface
func (c *PulsatrixCharger) CurrentPower() (float64, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if !c.valid() {
		c.handleError(c.Enable(false))
		return 0, fmt.Errorf(api.ErrOutdated.Error())
	}
	return float64(c.LastActivePower), nil
}

// Currents implements the api.PhaseCurrents interface
func (c *PulsatrixCharger) Currents() (float64, float64, float64, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	currents := c.PhaseAmperage
	if len(currents) < 3 || !c.valid() {
		c.handleError(c.Enable(false))
		return 0, 0, 0, fmt.Errorf(api.ErrOutdated.Error())
	}
	return currents[0], currents[1], currents[2], nil
}

// Voltages implements the api.PhaseVoltages interface
func (c *PulsatrixCharger) Voltages() (float64, float64, float64, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	voltages := c.PhaseVoltage
	if len(voltages) < 3 || !c.valid() {
		c.handleError(c.Enable(false))
		return 0, 0, 0, fmt.Errorf(api.ErrOutdated.Error())
	}
	return voltages[0], voltages[1], voltages[2], nil
}

// Total Energy implements the api.MeterEnergy interface
func (c *PulsatrixCharger) TotalEnergy() (float64, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if !c.valid() {
		c.handleError(c.Enable(false))
		return 0, fmt.Errorf(api.ErrOutdated.Error())
	}
	return float64(c.EnergyImported), nil
}
