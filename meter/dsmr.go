package meter

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/basvdlei/gotsmart/crc16"
	"github.com/basvdlei/gotsmart/dsmr"
	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// github.com/basvdlei/gotsmart package is subject to the following license:

// BSD 3-Clause License

// Copyright (c) 2017, Bas van der Lei
// All rights reserved.

// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:

// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.

// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.

// * Neither the name of the copyright holder nor the names of its
//   contributors may be used to endorse or promote products derived from
//   this software without specific prior written permission.

// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
// FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
// DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
// SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
// CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
// OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// Dsmr meter implementation
type Dsmr struct {
	mu      sync.Mutex
	addr    string
	energy  string
	timeout time.Duration
	frame   dsmr.Frame
	updated time.Time
}

var (
	currentObis     = []string{"1-0:31.7.0", "1-0:51.7.0", "1-0:71.7.0"}
	powerExportObis = []string{"1-0:22.7.0", "1-0:42.7.0", "1-0:62.7.0"}
)

func init() {
	registry.Add("dsmr", NewDsmrFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateDsmr -b api.Meter -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)"

// NewDsmrFromConfig creates a DSMR meter from generic config
func NewDsmrFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI     string
		Energy  string
		Timeout time.Duration
	}{
		Timeout: 15 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewDsmr(cc.URI, cc.Energy, cc.Timeout)
}

// NewDsmr creates DSMR meter
func NewDsmr(uri, energy string, timeout time.Duration) (api.Meter, error) {
	m := &Dsmr{
		addr:    uri,
		energy:  energy,
		timeout: timeout,
	}

	done := make(chan struct{}, 1)
	conn, err := m.connect()
	if err != nil {
		return nil, err
	}

	go m.run(conn, done)

	// wait for initial value
	select {
	case <-done:
	case <-time.NewTimer(timeout).C:
		return nil, os.ErrDeadlineExceeded
	}

	// decorate energy reading
	var totalEnergy func() (float64, error)
	if energy != "" {
		totalEnergy = m.totalEnergy
	}

	// decorate currents
	var currents func() (float64, float64, float64, error)

	for _, obis := range currentObis {
		_, err = m.get(obis)
		if err != nil {
			break
		}
	}

	if err == nil {
		currents = m.currents
	}

	return decorateDsmr(m, totalEnergy, currents), nil
}

// based on https://github.com/basvdlei/gotsmart/blob/master/gotsmart.go
func (m *Dsmr) run(conn net.Conn, done chan struct{}) {
	log := util.NewLogger("dsmr")
	backoff := backoff.NewExponentialBackOff()
	backoff.InitialInterval = time.Second
	backoff.MaxInterval = 5 * time.Minute

	handle := func(op string, err error) {
		log.ERROR.Printf("%s: %v", op, err)
		if err == io.EOF ||
			errors.Is(err, io.ErrUnexpectedEOF) ||
			errors.Is(err, net.ErrClosed) {
			conn.Close() // closing on nil socket is safe
			conn = nil
		}
	}

	reader := bufio.NewReader(conn)

	for {
		if conn == nil {
			var err error
			conn, err = m.connect()
			if err != nil {
				handle("connect", err)
				sleep := backoff.NextBackOff().Truncate(time.Second)
				log.DEBUG.Printf("next attempt after: %v", sleep)
				time.Sleep(sleep)
				continue
			}

			reader.Reset(conn)
		}

		backoff.Reset()
		if b, err := reader.Peek(1); err == nil {
			if string(b) != "/" {
				log.DEBUG.Printf("ignoring garbage character: %c\n", b)
				_, _ = reader.ReadByte()
				continue
			}
		} else {
			handle("peek", err)
			continue
		}

		frame, err := reader.ReadBytes('!')
		if err != nil {
			handle("read", err)
			continue
		}

		bcrc, err := reader.ReadBytes('\n')
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

func (m *Dsmr) connect() (net.Conn, error) {
	dialer := net.Dialer{Timeout: request.Timeout}

	conn, err := dialer.Dial("tcp", m.addr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (m *Dsmr) get(id string) (float64, error) {
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

// CurrentPower implements the api.Meter interface
func (m *Dsmr) CurrentPower() (float64, error) {
	bezug, err := m.get("1-0:1.7.0")

	var lief float64
	if err == nil {
		lief, err = m.get("1-0:2.7.0")
	}

	return (bezug - lief) * 1e3, err
}

// totalEnergy implements the api.MeterEnergy interface
func (m *Dsmr) totalEnergy() (float64, error) {
	return m.get(m.energy)
}

// currents implements the api.PhaseCurrents interface
func (m *Dsmr) currents() (float64, float64, float64, error) {
	var res [3]float64

	for i := 0; i < 3; i++ {
		var err error
		if res[i], err = m.get(currentObis[i]); err != nil {
			return 0, 0, 0, err
		}

		// correct import/export sign
		if f, err := m.get(powerExportObis[i]); err != nil {
			return 0, 0, 0, err
		} else if f > 0 {
			res[i] = -res[i]
		}
	}

	return res[0], res[1], res[2], nil
}
