package meter

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/basvdlei/gotsmart/crc16"
	"github.com/basvdlei/gotsmart/dsmr"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Dsmr meter implementation
type Dsmr struct {
	mu           sync.Mutex
	addr         string
	energyObis   string
	currentsObis []string
	timeout      time.Duration
	frame        dsmr.Frame
	updated      time.Time
}

func init() {
	registry.Add("dsmr", NewDsmrFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateDsmr -b api.Meter -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.MeterCurrent,Currents,func() (float64, float64, float64, error)"

// NewDsmrFromConfig creates a DSMR meter from generic config
func NewDsmrFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI      string
		Energy   string
		Currents []string
		Timeout  time.Duration
	}{
		Timeout: 15 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewDsmr(cc.URI, cc.Energy, cc.Currents, cc.Timeout)
}

// NewDsmr creates DSMR meter
func NewDsmr(uri, energy string, currents []string, timeout time.Duration) (api.Meter, error) {
	m := &Dsmr{
		addr:         uri,
		energyObis:   energy,
		currentsObis: currents,
		timeout:      timeout,
	}

	done := make(chan struct{}, 1)
	conn, err := m.connect()
	if err == nil {
		go m.run(conn, done)
	}

	// decorate energy reading
	var totalEnergy func() (float64, error)
	if energy != "" {
		totalEnergy = m.totalEnergy
	}

	// decorate currents
	var current func() (float64, float64, float64, error)
	if len(currents) > 0 {
		if len(currents) != 3 {
			return nil, errors.New("need 3 currents")
		}

		current = m.currents
	}

	// wait for initial value
	select {
	case <-done:
	case <-time.NewTimer(timeout).C:
		return nil, api.ErrTimeout
	}

	return decorateDsmr(m, totalEnergy, current), nil
}

// based on https://github.com/basvdlei/gotsmart/blob/master/gotsmart.go
func (m *Dsmr) run(conn *bufio.Reader, done chan struct{}) {
	log := util.NewLogger("dsmr")

	handle := func(op string, err error) {
		log.ERROR.Printf("%s: %v", op, err)
		if errors.Is(err, net.ErrClosed) {
			conn = nil
		}
	}

	for {
		if conn == nil {
			var err error
			conn, err = m.connect()
			if err != nil {
				handle("connect", err)
				time.Sleep(time.Second)
				continue
			}
		}

		if b, err := conn.Peek(1); err == nil {
			if string(b) != "/" {
				log.DEBUG.Printf("ignoring garbage character: %c\n", b)
				_, _ = conn.ReadByte()
				continue
			}
		} else {
			handle("peek", err)
			continue
		}

		frame, err := conn.ReadBytes('!')
		if err != nil {
			handle("read", err)
			continue
		}

		bcrc, err := conn.ReadBytes('\n')
		if err != nil {
			handle("read", err)
			continue
		}

		log.TRACE.Printf("read: %s\n", frame)

		// Check CRC
		mcrc := strings.ToUpper(strings.TrimSpace(string(bcrc)))
		crc := fmt.Sprintf("%04X", crc16.Checksum(frame))
		if mcrc != crc {
			log.ERROR.Printf("crc mismatch: %q != %q\n", mcrc, crc)
			continue
		}

		dsmrFrame, err := dsmr.ParseFrame(string(frame))
		if err != nil {
			log.ERROR.Printf("could not parse frame: %v\n", err)
			continue
		}

		m.mu.Lock()
		m.frame = dsmrFrame
		m.updated = time.Now()
		m.mu.Unlock()

		select {
		case done <- struct{}{}:
		default:
		}
	}
}

func (m *Dsmr) connect() (*bufio.Reader, error) {
	dialer := net.Dialer{Timeout: request.Timeout}

	conn, err := dialer.Dial("tcp", m.addr)
	if err != nil {
		return nil, err
	}

	return bufio.NewReader(conn), nil
}

func (m *Dsmr) get(id string) (dsmr.DataObject, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if time.Since(m.updated) > m.timeout {
		return dsmr.DataObject{}, api.ErrTimeout
	}

	res, ok := m.frame.Objects[id]
	if !ok {
		return dsmr.DataObject{}, fmt.Errorf("%w: %s", api.ErrNotAvailable, id)
	}

	return res, nil
}

// CurrentPower implements the api.Meter interface
func (m *Dsmr) CurrentPower() (float64, error) {
	bezug, err := m.get("1-0:1.7.0")
	if err != nil {
		return 0, err
	}

	lief, err := m.get("1-0:2.7.0")
	if err != nil {
		return 0, err
	}

	sum, err := strconv.ParseFloat(bezug.Value, 64)
	if err == nil {
		var f float64
		f, err = strconv.ParseFloat(lief.Value, 64)
		sum -= f
	}

	return sum * 1e3, err
}

// totalEnergy implements the api.MeterEnergy interface
func (m *Dsmr) totalEnergy() (float64, error) {
	res, err := m.get(m.energyObis)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(res.Value, 64)
}

// currents implements the api.MeterCurrent interface
func (m *Dsmr) currents() (float64, float64, float64, error) {
	var res [3]float64

	for i := 0; i < 3; i++ {
		val, err := m.get(m.currentsObis[i])
		if err != nil {
			return 0, 0, 0, err
		}

		res[i], err = strconv.ParseFloat(val.Value, 64)
		if err != nil {
			return 0, 0, 0, err
		}
	}

	return res[0], res[1], res[2], nil
}
