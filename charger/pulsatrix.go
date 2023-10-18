package charger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/gorilla/websocket"
)

// pulsatrix charger implementation
type PulsatrixCharger struct {
	conn              *websocket.Conn
	log               *util.Logger
	tryRead           int
	enState           bool
	path              string
	reconnecting      bool
	ConnectorStatus   string             `json:"connectorStatus"`
	VehicleStatus     string             `json:"vehicleStatus"`
	LastMeterValue    float64            `json:"lastMeterValue"`
	LastActivePower   float64            `json:"lastActivePower"`
	MeterStart        float64            `json:"meterStart"`
	State             string             `json:"state"`
	UsedPhasesSession string             `json:"usedPhasesSession"`
	PeakPhaseAmperage map[string]float64 `json:"peakPhaseAmperage"`
	AllocatedAmperage float64            `json:"allocatedAmperage"`
	CommandedAmperage float64            `json:"commandedAmperage"`
	AvailableAmperage float64            `json:"availableAmperage"`
	SignaledAmperage  float64            `json:"signaledAmperage"`
	//ChargeControllerStatus string             `json:"chargeControllerStatus"`
	//ID                     string             `json:"id"`
	//EffectiveAmperageLimit float64            `json:"effectiveAmperageLimit"`
	//StartedTime       	 int64              `json:"startedTime"`
}

func init() {
	registry.Add("pulsatrix", NewPulsatrixFromConfig)
}

// NewPulsatrixtFromConfig creates a pulsatrix charger from generic config
func NewPulsatrixFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Host  string
		Path  string
		Cache time.Duration
	}{
		Cache: time.Second,
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewPulsatrix(cc.Host, cc.Path)
}

// NewPulsatrix creates pulsatrix charger
func NewPulsatrix(hostname, path string) (*PulsatrixCharger, error) {
	wb := PulsatrixCharger{}
	err := wb.connectWs(hostname, path)
	if err != nil {
		log.Fatal("dial:", err)
	}
	wb.log = util.NewLogger("pulsatrix")
	return &wb, err
}

// ConnectWs connects to a pulsatrix charger websocket
func (c *PulsatrixCharger) connectWs(hostname, path string) error {
	u := url.URL{Scheme: "ws", Host: hostname, Path: path}
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		c.reconnectWs()
		return err
	}
	c.path = path
	c.enState = false
	c.conn = conn
	c.Enable(false)
	c.reconnecting = false
	go c.wsReader()
	return nil
}

// ReconnectWs reconnects to a pulsatrix charger websocket
func (c *PulsatrixCharger) reconnectWs() {
	c.reconnecting = true
	c.conn.Close()
	time.Sleep(60 * time.Second)
	c.log.INFO.Println("Reconnecting...")
	c.connectWs(c.conn.RemoteAddr().String(), c.path)
}

// WsReader runs a loop that reads messages from the websocket
func (c *PulsatrixCharger) wsReader() {
	var breaker bool
	go func() {
		for breaker == false {
			messageType, message, err := c.conn.ReadMessage()
			if err != nil {
				if c.tryRead < 3 {
					c.log.ERROR.Println("error reading message: ", err, " trying to read again")
					c.tryRead++
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
	return
}

// wsWrite writes a message to the websocket
func (c *PulsatrixCharger) wsWrite(message []byte) error {
	if !c.reconnecting {
		err := c.conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			fmt.Println("write error: ", err)
			return err
		}
		return nil
	} else {
		return fmt.Errorf("connection is reconnecting")
	}
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
		fmt.Println("messageType is not TextMessage")
	}
}

// Status implements the api.Charger interface
func (c *PulsatrixCharger) Status() (api.ChargeStatus, error) {
	status := c.VehicleStatus
	switch status {
	case "A":
		if c.enState {
			c.Enable(false)
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
		return api.StatusNone, nil
	}
}

// Enabled implements the api.Charger interface
func (c *PulsatrixCharger) Enabled() (bool, error) {
	//avaAmp := c.AvailableAmperage
	EnStatus := c.VehicleStatus
	if EnStatus == "C" || EnStatus == "D" || c.enState {
		return true, nil
	} else {
		return false, nil
	}
}

// Enable implements the api.Charger interface
func (c *PulsatrixCharger) Enable(enable bool) error {
	fmt.Println("Enable : ", enable)
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
	if !c.enState {
		c.Enable(true)
	}
	res := strconv.FormatFloat(current, 'f', 10, 64)
	return c.wsWrite([]byte("setCurrentLimit\n" + res))
}

// GetMaxCurrent implements the api.CurrentGetter interface
func (c *PulsatrixCharger) GetMaxCurrent() (float64, error) {
	//c.log.INFO.Println("AllocatedAmperage: ", c.AllocatedAmperage, "AvailableAmperage: ", c.AvailableAmperage, "SignaledAmperage: ", c.SignaledAmperage, "CommandedAmperage: ", c.CommandedAmperage)
	return float64(c.AllocatedAmperage), nil
}

// CurrentPower implements the api.PhaseCurrents interface
func (c *PulsatrixCharger) CurrentPower() (float64, error) {
	return float64(c.LastActivePower), nil
}

// ChargedEnergy implements the api.ChargeRater interface
func (c *PulsatrixCharger) ChargedEnergy() (float64, error) {
	var vStart = c.MeterStart
	var vLast = c.LastMeterValue
	return float64(vLast - vStart), nil
}

// StartCharge implements the api.VehicleChargeController interface
func (c *PulsatrixCharger) StartCharge() error {
	c.log.INFO.Println("StartCharge function called")
	return c.wsWrite([]byte("setCurrentLimit\n6"))
}

// StopCharge implements the api.VehicleChargeController interface
func (c *PulsatrixCharger) StopCharge() error {
	c.log.INFO.Println("StopCharge function called")
	return c.wsWrite([]byte("setCurrentLimit\n0"))
}

//remove comment after implementation of phase change in SECC firmware
/*
func (c *PulsatrixCharger) Phases1p3p(phases int) error {
	fmt.Println("Phases1p3p function called with: ", phases)
	if phases == 3 {
		err := c.wsWrite([]byte("Phases1p3\n1"))
		if err != nil {
			return err
		}
	} else {
		err := c.wsWrite([]byte("Phases1p3\n1"))
		if err != nil {
			return err
		}
	}
	return nil
}
*/
