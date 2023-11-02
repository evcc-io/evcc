package charger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/gorilla/websocket"
)

// pulsatrix charger implementation
type PulsatrixCharger struct {
	conn         *websocket.Conn
	log          *util.Logger
	tryRead      int
	enState      bool
	path         string
	reconnecting int
	signaledAmp  float64
	mutex        sync.Mutex
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
	cc := struct {
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
	err := wb.connectWs(hostname)
	if err != nil {
		return nil, err
	}
	return &wb, nil
}

// ConnectWs connects to a pulsatrix SECC websocket
func (c *PulsatrixCharger) connectWs(hostname string) error {
	u := url.URL{Scheme: "ws", Host: hostname, Path: "/api/ws"}
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		if strErr := fmt.Sprintf("%s", err); strErr == "websocket: bad handshake" {
			c.log.ERROR.Println("bad handshake. Make sure the IP and port are correct")
		} else {
			c.log.ERROR.Println("error:", err)
			c.reconnectWs()
		}
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			c.log.ERROR.Println("Recovered from panic:", r)
		}
	}()
	if c.reconnecting > 0 {
		c.log.WARN.Printf("connection reestablished\n")
		c.reconnecting = 0
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
	for {
		if c.VehicleStatus == "A" {
			c.handleError(c.Enable(false))
			time.Sleep(2 * time.Minute)
		} else {
			if c.enState {
				c.handleError(c.Enable(true))
				c.handleError(c.MaxCurrentMillis(c.signaledAmp))
			} else {
				c.handleError(c.Enable(false))
			}
			time.Sleep(30 * time.Second)
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
	c.reconnecting++
	c.conn.Close()
	var waitTime time.Duration
	if c.reconnecting == 1 {
		c.log.WARN.Printf("lost connection, trying to reconnect\n")
		waitTime = 45 * time.Second
	} else if c.reconnecting <= 5 {
		waitTime = time.Duration(c.reconnecting) * time.Minute
		c.log.WARN.Printf("trying to reconnect in %d minutes...\n", c.reconnecting)
	} else {
		waitTime = 5 * time.Minute
		c.log.WARN.Printf("trying to reconnect in 5 minutes...\n")
	}

	time.Sleep(waitTime)
	c.handleError(c.connectWs(c.conn.RemoteAddr().String()))
}

// WsReader runs a loop that reads messages from the websocket
func (c *PulsatrixCharger) wsReader() {
	var breaker bool
	go func() {
		for !breaker {
			messageType, message, err := c.conn.ReadMessage()
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
func (c *PulsatrixCharger) parseWsMessage(messageType int, message []byte) {
	if messageType == websocket.TextMessage {
		b := bytes.Replace(message, []byte(":NaN"), []byte(":null"), -1)
		stadtIdx := bytes.IndexByte(b, '{')
		err := json.Unmarshal(b[stadtIdx:], c)
		if err != nil {
			return
		}
	} else {
		c.log.WARN.Println("Websocket message is not a TextMessage")
	}
}

// wsWrite writes a message to the websocket
func (c *PulsatrixCharger) wsWrite(message []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.reconnecting != 0 {
		return fmt.Errorf("reconnecting...")
	}
	c.handleError(c.conn.WriteMessage(websocket.TextMessage, message))

	return nil
}

// Status implements the api.Charger interface
func (c *PulsatrixCharger) Status() (api.ChargeStatus, error) {
	status := c.VehicleStatus
	switch status {
	case "A":
		if c.enState {
			c.handleError(c.Enable(false))
		}
		return api.StatusA, nil
	case "B":
		return api.StatusB, nil
	case "C":
		return api.StatusC, nil
	case "D":
		return api.StatusD, nil
	case "E":
		return api.StatusE, nil
	case "F":
		return api.StatusF, nil
	default:
		fmt.Println("error: API-Status is not defined")
		return api.StatusNone, nil

	}
}

// Enabled implements the api.Charger interface
func (c *PulsatrixCharger) Enabled() (bool, error) {
	if c.enState {
		return true, nil
	} else {
		return false, nil
	}
}

// Enable implements the api.Charger interface
func (c *PulsatrixCharger) Enable(enable bool) error {
	var state string
	if enable {
		c.enState = true
		state = "true"
	} else {
		c.enState = false
		state = "false"
	}
	return c.wsWrite([]byte("setEnabled\n" + state))
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
	currents := c.PhaseAmperage
	return currents[0], currents[1], currents[2], nil
}

// Voltages implements the api.PhaseVoltages interface
func (c *PulsatrixCharger) Voltages() (float64, float64, float64, error) {
	voltages := c.PhaseVoltage
	return voltages[0], voltages[1], voltages[2], nil
}

// Total Energy implements the api.MeterEnergy interface
func (c *PulsatrixCharger) TotalEnergy() (float64, error) {
	return float64(c.EnergyImported), nil
}
