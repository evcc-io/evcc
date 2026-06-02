package meter

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/basvdlei/gotsmart/crc16"
	"github.com/basvdlei/gotsmart/dsmr"
	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/gorilla/websocket"
)

// Homey meter implementation (Homey Energy Dongle via WebSocket + DSMR)
type Homey struct {
	implement.Caps
	mu      sync.Mutex
	uri     string
	timeout time.Duration
	frame   dsmr.Frame
	updated time.Time
}

func init() {
	registry.Add("homey", NewHomeyFromConfig)
}

// NewHomeyFromConfig creates a Homey meter from generic config
func NewHomeyFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		URI     string
		Timeout time.Duration
	}{
		Timeout: 15 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewHomey(cc.URI, cc.Timeout)
}

// NewHomey creates a Homey meter
func NewHomey(uri string, timeout time.Duration) (api.Meter, error) {
	// A bare hostname (e.g. "192.168.1.100") has no scheme and no "//", so
	// url.Parse puts everything into Path instead of Host. Prepend "//" to
	// force correct host parsing before we add the ws scheme.
	if !strings.Contains(uri, "://") && !strings.HasPrefix(uri, "//") {
		uri = "//" + uri
	}
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "http" || u.Scheme == "" {
		u.Scheme = "ws"
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = "/ws"
	}

	m := &Homey{
		Caps:    implement.New(),
		uri:     u.String(),
		timeout: timeout,
	}

	done := make(chan struct{}, 1)
	go m.run(done)

	select {
	case <-done:
	case <-time.NewTimer(timeout).C:
		return nil, os.ErrDeadlineExceeded
	}

	// DSMR P1 always carries import and export energy registers — register both unconditionally.
	implement.Has(m, implement.MeterEnergy(m.totalEnergy))
	implement.Has(m, implement.MeterReturnEnergy(m.totalReturnEnergy))

	// decorate currents if all three phases are available
	err = nil
	for _, obis := range currentObis {
		_, err = m.get(obis)
		if err != nil {
			break
		}
	}
	if err == nil {
		implement.Has(m, implement.PhaseCurrents(m.currents))
	}

	return m, nil
}

// wsReader wraps a gorilla WebSocket connection as an io.Reader by chaining
// per-message readers so the DSMR stream parser sees a continuous byte stream.
type wsReader struct {
	conn   *websocket.Conn
	reader io.Reader
}

func (r *wsReader) Read(p []byte) (int, error) {
	for {
		if r.reader != nil {
			n, err := r.reader.Read(p)
			if n > 0 {
				return n, nil
			}
			if err != io.EOF {
				return 0, err
			}
			r.reader = nil
		}

		_, reader, err := r.conn.NextReader()
		if err != nil {
			return 0, err
		}
		r.reader = reader
	}
}

func (m *Homey) connect() (*websocket.Conn, error) {
	conn, _, err := websocket.DefaultDialer.Dial(m.uri, nil)
	if err != nil {
		return nil, err
	}
	conn.SetPongHandler(func(string) error { return nil })
	return conn, nil
}

func (m *Homey) run(done chan struct{}) {
	log := util.NewLogger("homey")
	bo := backoff.NewExponentialBackOff(backoff.WithMaxInterval(5*time.Minute), backoff.WithMaxElapsedTime(0))

	var conn *websocket.Conn

	handle := func(op string, err error) {
		log.ERROR.Printf("%s: %v", op, err)
		if conn != nil {
			conn.Close()
			conn = nil
		}
	}

	for {
		if conn == nil {
			var err error
			conn, err = m.connect()
			if err != nil {
				handle("connect", err)
				time.Sleep(max(bo.NextBackOff(), time.Second))
				continue
			}
			bo.Reset()
		}

		reader := bufio.NewReader(&wsReader{conn: conn})

		for {
			b, err := reader.Peek(1)
			if err != nil {
				handle("peek", err)
				break
			}
			if string(b) != "/" {
				log.DEBUG.Printf("ignoring garbage character: %c\n", b[0])
				_, _ = reader.ReadByte()
				continue
			}

			frame, err := reader.ReadBytes('!')
			if err != nil {
				handle("read frame", err)
				break
			}

			bcrc, err := reader.ReadBytes('\n')
			if err != nil {
				handle("read crc", err)
				break
			}

			log.TRACE.Printf("read: %s", frame)

			mcrc := strings.ToUpper(strings.TrimSpace(string(bcrc)))
			crc := fmt.Sprintf("%04X", crc16.Checksum(frame))
			if mcrc != crc {
				log.ERROR.Printf("crc mismatch: %q != %q", mcrc, crc)
				continue
			}

			dsmrFrame, err := dsmr.ParseFrame(string(frame))
			if err != nil {
				log.ERROR.Printf("could not parse frame: %v", err)
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
}

func (m *Homey) get(id string) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if time.Since(m.updated) > m.timeout {
		return 0, os.ErrDeadlineExceeded
	}

	res, ok := m.frame.Objects[id]
	if !ok {
		return 0, fmt.Errorf("%w: %s", api.ErrNotAvailable, id)
	}

	return strconv.ParseFloat(res.Value, 64)
}

func (m *Homey) sumPhases(obis [3]string) (float64, error) {
	var sum float64
	for _, o := range obis {
		f, err := m.get(o)
		if err != nil {
			return 0, err
		}
		sum += f
	}
	return sum, nil
}

// CurrentPower implements the api.Meter interface
func (m *Homey) CurrentPower() (float64, error) {
	bezug, err1 := m.get("1-0:1.7.0")
	if errors.Is(err1, api.ErrNotAvailable) {
		if f, err := m.sumPhases([3]string{"1-0:21.7.0", "1-0:41.7.0", "1-0:61.7.0"}); err == nil {
			bezug = f
			err1 = nil
		}
	}

	lief, err2 := m.get("1-0:2.7.0")
	if errors.Is(err2, api.ErrNotAvailable) {
		if f, err := m.sumPhases([3]string{"1-0:22.7.0", "1-0:42.7.0", "1-0:62.7.0"}); err == nil {
			lief = f
			err2 = nil
		}
	}

	// allow one value to be missing
	if err1 == nil && errors.Is(err2, api.ErrNotAvailable) || err2 == nil && errors.Is(err1, api.ErrNotAvailable) {
		err1 = nil
		err2 = nil
	}

	return (bezug - lief) * 1e3, errors.Join(err1, err2)
}

// totalEnergy implements the api.MeterEnergy interface by summing import tariff 1 + tariff 2
func (m *Homey) totalEnergy() (float64, error) {
	return m.sumObis([2]string{"1-0:1.8.1", "1-0:1.8.2"})
}

// totalReturnEnergy implements the api.MeterReturnEnergy interface by summing export tariff 1 + tariff 2
func (m *Homey) totalReturnEnergy() (float64, error) {
	return m.sumObis([2]string{"1-0:2.8.1", "1-0:2.8.2"})
}

func (m *Homey) sumObis(obis [2]string) (float64, error) {
	var total float64
	for _, o := range obis {
		f, err := m.get(o)
		if err != nil {
			return 0, err
		}
		total += f
	}
	return total, nil
}

// currents implements the api.PhaseCurrents interface
func (m *Homey) currents() (float64, float64, float64, error) {
	var res [3]float64

	for i := range res {
		var err error
		if res[i], err = m.get(currentObis[i]); err != nil {
			return 0, 0, 0, err
		}

		if f, err := m.get(powerExportObis[i]); err != nil {
			return 0, 0, 0, err
		} else if f > 0 {
			res[i] = -res[i]
		}
	}

	return res[0], res[1], res[2], nil
}
