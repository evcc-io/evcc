package charger

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/charger/keba"
	"github.com/andig/evcc/util"
)

// https://www.keba.com/file/downloads/e-mobility/KeContact_P20_P30_UDP_ProgrGuide_en.pdf

const (
	udpTimeout = time.Second
	kebaPort   = "7090"
)

// Keba is an api.Charger implementation with configurable getters and setters.
type Keba struct {
	log     *util.Logger
	conn    string
	timeout time.Duration
	recv    chan keba.UDPMsg
}

// NewKebaFromConfig creates a new configurable charger
func NewKebaFromConfig(log *util.Logger, other map[string]interface{}) api.Charger {
	cc := struct {
		URI     string
		Timeout time.Duration
	}{}
	util.DecodeOther(log, other, &cc)

	return NewKeba(cc.URI, cc.Timeout)
}

// NewKeba creates a new charger
func NewKeba(conn string, timeout time.Duration) api.Charger {
	log := util.NewLogger("keba")

	if keba.Instance == nil {
		keba.Instance = keba.New(log, fmt.Sprintf(":%s", kebaPort))
	}

	// add default port
	if _, _, err := net.SplitHostPort(conn); err != nil {
		conn = fmt.Sprintf("%s:%s", conn, kebaPort)
	}

	if timeout == 0 {
		timeout = udpTimeout
	}

	c := &Keba{
		log:     log,
		conn:    conn,
		timeout: timeout,
		recv:    make(chan keba.UDPMsg),
	}

	keba.Instance.Subscribe(conn, c.recv)

	return c
}

func (c *Keba) send(msg string) error {
	raddr, err := net.ResolveUDPAddr("udp", c.conn)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return err
	}

	defer conn.Close()

	_, err = io.Copy(conn, strings.NewReader(msg))
	return err
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

	if err := c.send(msg); err != nil {
		return err
	}

	select {
	case resp := <-resC:
		if report == 0 {
			// use reflection to write to simple string
			rv := reflect.ValueOf(res)
			if rv.Kind() != reflect.Ptr || rv.IsNil() || rv.Elem().Kind() != reflect.String {
				return fmt.Errorf("invalid type: %s", reflect.TypeOf(res))
			}

			rv.Elem().SetString(string(resp.Message))
			return nil
		}
		return json.Unmarshal(resp.Message, &res)
	case err := <-errC:
		return err
	}
}

// Status implements the Charger.Status interface
func (c *Keba) Status() (api.ChargeStatus, error) {
	var kr keba.Report2
	err := c.roundtrip("report 2", 2, &kr)
	if err != nil {
		return api.StatusA, err
	}

	if kr.Plug == 0 {
		return api.StatusA, nil
	}
	if kr.State == 2 {
		return api.StatusB, nil
	}
	if kr.State == 3 {
		return api.StatusC, nil
	}

	return api.StatusA, fmt.Errorf("unexpected status: %v", kr)
}

// Enabled implements the Charger.Enabled interface
func (c *Keba) Enabled() (bool, error) {
	var kr keba.Report2
	err := c.roundtrip("report 2", 2, &kr)
	if err != nil {
		return false, err
	}

	return kr.EnableSys == 1 || kr.EnableUser == 1, nil
}

// Enable implements the Charger.Enable interface
func (c *Keba) Enable(enable bool) error {
	var d int
	if enable {
		d = 1
	}

	var resp string
	err := c.roundtrip(fmt.Sprintf("ena %d", d), 0, &resp)
	if err != nil {
		return err
	}

	if string(resp) == keba.OK {
		return nil
	}

	return fmt.Errorf("ena unexpected response: %s", resp)
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (c *Keba) MaxCurrent(current int64) error {
	var resp string
	err := c.roundtrip(fmt.Sprintf("curr %d", 1000*current), 0, &resp)
	if err != nil {
		return err
	}

	if resp == keba.OK {
		return nil
	}

	return fmt.Errorf("curr unexpected response: %s", resp)
}

// CurrentPower implements the Meter interface
func (c *Keba) CurrentPower() (float64, error) {
	var kr keba.Report3
	err := c.roundtrip("report 3", 3, &kr)

	// mW to W
	return float64(kr.P) / 1e3, err
}

// TotalEnergy implements the MeterEnergy interface
func (c *Keba) TotalEnergy() (float64, error) {
	var kr keba.Report3
	err := c.roundtrip("report 3", 3, &kr)

	// mW to W
	return float64(kr.ETotal) / 1e4, err
}

// ChargedEnergy implements the ChargeRater interface
func (c *Keba) ChargedEnergy() (float64, error) {
	var kr keba.Report3
	err := c.roundtrip("report 3", 3, &kr)

	// 0,1Wh to kWh
	return float64(kr.EPres) / 1e4, err
}

// Currents implements the MeterCurrents interface
func (c *Keba) Currents() (float64, float64, float64, error) {
	var kr keba.Report3
	err := c.roundtrip("report 3", 3, &kr)

	// 1mA to A
	return float64(kr.I1) / 1e3, float64(kr.I2) / 1e3, float64(kr.I3) / 1e3, err
}
