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
type Pulsatrix struct {
	log     *util.Logger
	mu      sync.Mutex
	conn    *websocket.Conn
	uri     string
	enabled bool
	current float64
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

	return &wb, wb.connectWs()
}

// ConnectWs connects to a pulsatrix SECC websocket
func (c *Pulsatrix) connectWs() error {
	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

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

	c.conn = conn

	// ensure evcc and SECC are in sync
	if err := c.Enable(false); err != nil {
		c.log.ERROR.Println(err)
	}
	go c.wsReader()
	go c.heartbeat()

	return nil
}

// ReconnectWs reconnects to a pulsatrix SECC websocket
func (c *Pulsatrix) reconnectWs() {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 30 * time.Second
	bo.MaxInterval = 5 * time.Minute

	if err := backoff.RetryNotify(c.connectWs, bo, func(err error, time time.Duration) {
		c.log.WARN.Printf("trying to reconnect in %v...\n", time)
	}); err != nil {
		c.log.ERROR.Println("RetryNotify: ", err)
	}
}

// WsReader runs a loop that reads messages from the websocket
func (c *Pulsatrix) wsReader() {
	_, err := c.data.Get()
	for ok := true; ok; ok = err != nil {
		ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
		defer cancel()

		messageType, message, err := c.conn.Read(ctx)
		if err != nil {
			c.log.ERROR.Println("error reading message:", err)
			break
		} else {
			c.parseWsMessage(messageType, message)
		}
	}

	c.mu.Lock()
	c.conn.Close(websocket.StatusNormalClosure, "Reconnecting")
	c.conn = nil
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
		idx := bytes.IndexByte(b, '{')

		val, _ := c.data.Get()
		if err := json.Unmarshal(b[idx:], &val); err != nil {
			c.log.WARN.Println(err)
		} else {
			c.data.Set(val)
		}
	}
}

// Heartbeat sends a heartbeat to the pulsatrix SECC
func (c *Pulsatrix) heartbeat() {
	for range time.Tick(3 * time.Minute) {
		if err := c.Enable(c.enabled); err != nil {
			c.log.ERROR.Println(err)
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
	err := c.write("setCurrentLimit\n" + strconv.FormatFloat(current, 'f', 10, 64))
	if err == nil {
		c.current = current
	}
	return err
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
