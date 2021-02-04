package charger

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/charger/keba"
	"github.com/andig/evcc/util"
)

// https://www.keba.com/file/downloads/e-mobility/KeContact_P20_P30_UDP_ProgrGuide_en.pdf

const (
	udpTimeout = time.Second
)

// RFID contains access credentials
type RFID struct {
	Tag string
}

// Keba is an api.Charger implementation with configurable getters and setters.
type Keba struct {
	log     *util.Logger
	conn    string
	rfid    RFID
	timeout time.Duration
	recv    chan keba.UDPMsg
	sender  *keba.Sender
}

func init() {
	registry.Add("keba", NewKebaFromConfig)
}

// NewKebaFromConfig creates a new configurable charger
func NewKebaFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI     string
		Serial  string
		Timeout time.Duration
		RFID    RFID
	}{
		Timeout: udpTimeout,
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewKeba(cc.URI, cc.Serial, cc.RFID, cc.Timeout)
}

// NewKeba creates a new charger
func NewKeba(uri, serial string, rfid RFID, timeout time.Duration) (api.Charger, error) {
	log := util.NewLogger("keba")

	if keba.Instance == nil {
		var err error
		keba.Instance, err = keba.New(log)
		if err != nil {
			return nil, err
		}
	}

	// add default port
	conn := util.DefaultPort(uri, keba.Port)
	sender, err := keba.NewSender(log, conn)

	c := &Keba{
		log:     log,
		conn:    conn,
		rfid:    rfid,
		timeout: timeout,
		recv:    make(chan keba.UDPMsg),
		sender:  sender,
	}

	// use serial to subscribe if defined for docker scenarios
	if serial == "" {
		serial = conn
	}

	keba.Instance.Subscribe(serial, c.recv)

	return c, err
}

func (c *Keba) receive(report int, resC chan<- keba.UDPMsg, errC chan<- error, closeC <-chan struct{}) {
	t := time.NewTimer(c.timeout)
	defer close(resC)
	defer close(errC)
	for {
		select {
		case msg := <-c.recv:
			// matching result message
			if msg.Report == nil && report == 0 {
				resC <- msg
				return
			}
			// matching report id
			if msg.Report != nil && report == msg.Report.ID {
				resC <- msg
				return
			}
		case <-t.C:
			errC <- errors.New("recv timeout")
			return
		case <-closeC:
			return
		}
	}
}

func (c *Keba) roundtrip(msg string, report int, res interface{}) error {
	resC := make(chan keba.UDPMsg)
	errC := make(chan error)
	closeC := make(chan struct{})

	defer close(closeC)

	go c.receive(report, resC, errC, closeC)

	// add report number to message and send
	if report > 0 {
		msg = fmt.Sprintf("%s %d", msg, report)
	}
	if err := c.sender.Send(msg); err != nil {
		return err
	}

	for {
		select {
		case resp := <-resC:
			if report == 0 {
				// use reflection to write to simple string
				rv := reflect.ValueOf(res)
				if rv.Kind() != reflect.Ptr || rv.IsNil() || rv.Elem().Kind() != reflect.String {
					return fmt.Errorf("invalid type: %s", reflect.TypeOf(res))
				}

				res := string(resp.Message)
				if res != keba.OK {
					continue
				}

				rv.Elem().SetString(res)
				return nil
			}
			return json.Unmarshal(resp.Message, &res)
		case err := <-errC:
			return err
		}
	}
}

// Status implements the Charger.Status interface
func (c *Keba) Status() (api.ChargeStatus, error) {
	var kr keba.Report2
	err := c.roundtrip("report", 2, &kr)
	if err != nil {
		return api.StatusA, err
	}

	if kr.AuthON == 1 && c.rfid.Tag == "" {
		c.log.WARN.Println("missing credentials for RFID authorization")
	}

	if kr.Plug < 5 {
		return api.StatusA, nil
	}
	if kr.State == 3 {
		return api.StatusC, nil
	}
	if kr.State != 4 {
		return api.StatusB, nil
	}

	return api.StatusA, fmt.Errorf("unexpected status: %+v", kr)
}

// Enabled implements the Charger.Enabled interface
func (c *Keba) Enabled() (bool, error) {
	var kr keba.Report2
	err := c.roundtrip("report", 2, &kr)
	if err != nil {
		return false, err
	}

	return kr.EnableSys == 1 || kr.EnableUser == 1, nil
}

// enableRFID sends RFID credentials to enable charge
func (c *Keba) enableRFID() error {
	// check if authorization required
	var kr keba.Report2
	if err := c.roundtrip("report", 2, &kr); err != nil {
		return err
	}
	if kr.AuthReq == 0 {
		return nil
	}

	// authorize
	var resp string
	if err := c.roundtrip(fmt.Sprintf("start %s", c.rfid.Tag), 0, &resp); err != nil {
		return err
	}

	return nil
}

// Enable implements the Charger.Enable interface
func (c *Keba) Enable(enable bool) error {
	if enable && c.rfid.Tag != "" {
		if err := c.enableRFID(); err != nil {
			return err
		}
	}

	var d int
	if enable {
		d = 1
	}

	// ignore result...
	var resp string
	if err := c.roundtrip(fmt.Sprintf("ena %d", d), 0, &resp); err != nil {
		return err
	}

	return nil
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (c *Keba) MaxCurrent(current int64) error {
	d := 1000 * current

	var resp string
	if err := c.roundtrip(fmt.Sprintf("curr %d", d), 0, &resp); err != nil {
		return err
	}

	return nil
}

// CurrentPower implements the Meter interface
func (c *Keba) CurrentPower() (float64, error) {
	var kr keba.Report3
	err := c.roundtrip("report", 3, &kr)

	// mW to W
	return float64(kr.P) / 1e3, err
}

// TotalEnergy implements the MeterEnergy interface
func (c *Keba) TotalEnergy() (float64, error) {
	var kr keba.Report3
	err := c.roundtrip("report", 3, &kr)

	// mW to W
	return float64(kr.ETotal) / 1e4, err
}

// ChargedEnergy implements the ChargeRater interface
func (c *Keba) ChargedEnergy() (float64, error) {
	var kr keba.Report3
	err := c.roundtrip("report", 3, &kr)

	// 0,1Wh to kWh
	return float64(kr.EPres) / 1e4, err
}

// Currents implements the MeterCurrents interface
func (c *Keba) Currents() (float64, float64, float64, error) {
	var kr keba.Report3
	err := c.roundtrip("report", 3, &kr)

	// 1mA to A
	return float64(kr.I1) / 1e3, float64(kr.I2) / 1e3, float64(kr.I3) / 1e3, err
}

// Diagnose implements the Diagnosis interface
func (c *Keba) Diagnose() {
	var kr keba.Report100
	if err := c.roundtrip("report", 100, &kr); err == nil {
		fmt.Printf("%+v\n", kr)
	}
}
