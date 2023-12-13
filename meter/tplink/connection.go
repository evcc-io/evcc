package tplink

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/evcc-io/evcc/util"
)

// Connection is the TP-Link connection
type Connection struct {
	log *util.Logger
	uri string
}

// NewConnection creates TP-Link charger
func NewConnection(uri string) (*Connection, error) {
	if uri == "" {
		return nil, errors.New("missing uri")
	}

	c := &Connection{
		log: util.NewLogger("tplink"),
		uri: net.JoinHostPort(uri, "9999"),
	}
	return c, nil
}

// ExecCmd executes an TP-Link Smart Home Protocol command and provides the response
func (d *Connection) ExecCmd(cmd string, res interface{}) error {
	// encode command message
	buf := bytes.NewBuffer([]byte{0, 0, 0, 0})
	var key byte = 171 // initialization vector
	for i := 0; i < len(cmd); i++ {
		key ^= cmd[i]
		_ = buf.WriteByte(key)
	}

	// write 4 bytes command length to start of buffer
	binary.BigEndian.PutUint32(buf.Bytes(), uint32(buf.Len()-4))

	// open connection via TP-Link Smart Home Protocol
	conn, err := net.DialTimeout("tcp", d.uri, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	// send command
	if _, err = buf.WriteTo(conn); err != nil {
		return err
	}

	// read response
	resp := make([]byte, 8192)
	len, err := conn.Read(resp)
	if err != nil {
		return err
	}

	// decode response message
	key = 171 // reset initialization vector
	for i := 4; i < len; i++ {
		dec := key ^ resp[i]
		key = resp[i]
		_ = buf.WriteByte(dec)
	}
	d.log.TRACE.Printf("recv: %s", buf.String())

	return json.Unmarshal(buf.Bytes(), res)
}

// CurrentPower implements the api.Meter interface
func (d *Connection) CurrentPower() (float64, error) {
	var res EmeterResponse
	if err := d.ExecCmd(`{"emeter":{"get_realtime":null}}`, &res); err != nil {
		return 0, err
	}

	if err := res.Emeter.GetRealtime.ErrCode; err != 0 {
		return 0, fmt.Errorf("get_realtime: %d", err)
	}

	power := res.Emeter.GetRealtime.PowerMw / 1000
	if power == 0 {
		power = res.Emeter.GetRealtime.Power
	}

	return power, nil
}

// TotalEnergy implements the api.MeterEnergy interface
func (d *Connection) TotalEnergy() (float64, error) {
	var res EmeterResponse
	if err := d.ExecCmd(`{"emeter":{"get_realtime":null}}`, &res); err != nil {
		return 0, err
	}

	if err := res.Emeter.GetRealtime.ErrCode; err != 0 {
		return 0, fmt.Errorf("get_realtime: %d", err)
	}

	energy := res.Emeter.GetRealtime.TotalWh / 1000
	if energy == 0 {
		energy = res.Emeter.GetRealtime.Total
	}

	return energy, nil
}
