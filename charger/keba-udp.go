package charger

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/keba"
	"github.com/evcc-io/evcc/util"
)

// https://www.keba.com/file/downloads/e-mobility/KeContact_P20_P30_UDP_ProgrGuide_en.pdf

const (
	udpTimeout = time.Second
)

// KebaUdp is an api.Charger implementation
type KebaUdp struct {
	log     *util.Logger
	conn    string
	rfid    keba.RFID
	timeout time.Duration
	recv    chan keba.UDPMsg
	sender  *keba.Sender
}

func init() {
	registry.Add("keba-udp", NewKebaUdpFromConfig)
}

//go:generate decorate -f decorateKebaUdp -b *KebaUdp -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)"

// NewKebaUdpFromConfig creates a new Keba UDP charger
func NewKebaUdpFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI     string
		Serial  string
		Timeout time.Duration
		RFID    keba.RFID
	}{
		Timeout: udpTimeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	k, err := NewKebaUdp(cc.URI, cc.Serial, cc.RFID, cc.Timeout)
	if err != nil {
		return nil, err
	}

	energy, err := k.totalEnergy()
	if err != nil {
		return nil, err
	}

	if energy > 0 {
		return decorateKebaUdp(k, k.currentPower, k.totalEnergy, k.currents), nil
	}

	return k, err
}

// NewKebaUdp creates a new charger
func NewKebaUdp(uri, serial string, rfid keba.RFID, timeout time.Duration) (*KebaUdp, error) {
	log := util.NewLogger("keba")

	instance, err := keba.Instance(log)
	if err != nil {
		return nil, err
	}

	// add default port
	conn := util.DefaultPort(uri, keba.Port)
	sender, err := keba.NewSender(log, conn)

	c := &KebaUdp{
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

	instance.Subscribe(serial, c.recv)

	return c, err
}

func (c *KebaUdp) receive(report int, resC chan<- keba.UDPMsg, errC chan<- error, closeC <-chan struct{}) {
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

func (c *KebaUdp) roundtrip(msg string, report int, res interface{}) error {
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

// Status implements the api.Charger interface
func (c *KebaUdp) Status() (api.ChargeStatus, error) {
	var kr keba.Report2
	err := c.roundtrip("report", 2, &kr)
	if err != nil {
		return api.StatusA, err
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

	return api.StatusNone, fmt.Errorf("invalid status: %+d", kr.State)
}

// Enabled implements the api.Charger interface
func (c *KebaUdp) Enabled() (bool, error) {
	var kr keba.Report2
	err := c.roundtrip("report", 2, &kr)
	if err != nil {
		return false, err
	}

	return kr.EnableSys == 1 || kr.EnableUser == 1, nil
}

// enableRFID sends RFID credentials to enable charge
func (c *KebaUdp) enableRFID() error {
	// check if authorization required
	var kr keba.Report2
	if err := c.roundtrip("report", 2, &kr); err != nil {
		return err
	}

	// no auth required
	if kr.AuthReq == 0 {
		return nil
	}

	// auth required but missing tag
	if c.rfid.Tag == "" {
		return errors.New("missing credentials for RFID authorization")
	}

	// authorize
	var resp string
	return c.roundtrip(fmt.Sprintf("start %s", c.rfid.Tag), 0, &resp)
}

// Enable implements the api.Charger interface
func (c *KebaUdp) Enable(enable bool) error {
	if enable {
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

// MaxCurrent implements the api.Charger interface
func (c *KebaUdp) MaxCurrent(current int64) error {
	d := 1000 * current

	var resp string
	if err := c.roundtrip(fmt.Sprintf("curr %d", d), 0, &resp); err != nil {
		return err
	}

	return nil
}

var _ api.ChargerEx = (*KebaUdp)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *KebaUdp) MaxCurrentMillis(current float64) error {
	d := int(1000 * current)

	var resp string
	if err := c.roundtrip(fmt.Sprintf("curr %d", d), 0, &resp); err != nil {
		return err
	}
	if resp != keba.OK {
		return fmt.Errorf("curr %d unexpected response: %s", d, resp)
	}

	return nil
}

// currentPower implements the api.Meter interface
func (c *KebaUdp) currentPower() (float64, error) {
	var kr keba.Report3
	err := c.roundtrip("report", 3, &kr)

	// mW to W
	return float64(kr.P) / 1e3, err
}

// totalEnergy implements the api.MeterEnergy interface
func (c *KebaUdp) totalEnergy() (float64, error) {
	var kr keba.Report3
	err := c.roundtrip("report", 3, &kr)

	// mW to W
	return float64(kr.ETotal) / 1e4, err
}

// currents implements the api.PhaseCurrents interface
func (c *KebaUdp) currents() (float64, float64, float64, error) {
	var kr keba.Report3
	err := c.roundtrip("report", 3, &kr)

	// 1mA to A
	return float64(kr.I1) / 1e3, float64(kr.I2) / 1e3, float64(kr.I3) / 1e3, err
}

var _ api.Identifier = (*KebaUdp)(nil)

// Identify implements the api.Identifier interface
func (c *KebaUdp) Identify() (string, error) {
	var kr keba.Report100
	err := c.roundtrip("report", 100, &kr)
	return kr.RFIDTag, err
}

var _ api.Diagnosis = (*KebaUdp)(nil)

// Diagnose implements the api.Diagnosis interface
func (c *KebaUdp) Diagnose() {
	var kr keba.Report100
	if err := c.roundtrip("report", 100, &kr); err == nil {
		fmt.Printf("%+v\n", kr)
	}
}
