package charger

/*
MIT License

Copyright (c) 2023-2025 pulsatrix gmbh
Copyright (c) 2019-2025 andig

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

/*
This module integrates the pulsatrix Supply Equipment Charge Controller (SECC)
with evcc.io enabling dynamic PV surplus charging. Communication is handled via
a bidirectional WebSocket connection, exchanging state data (e.g. vehicle
status, voltages, currents, energy counters) and sending control commands (e.g.
start/stop charging, current limits, phase switching) to the controller.

Robust operation is ensured by automatic detection and handling of connection
losses, with reconnection based on exponential backoff strategies. In addition
to real-time data exchange, periodic heartbeats maintain connectivity, while
connection statistics such as uptime, error counts, and reconnection attempts
are tracked.

For further details, see: https://docs.pulsatrix.com or https://pulsatrix.de
*/

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
)

const (
	dataTimeout       = 15 * time.Second
	heartbeatInterval = 3 * time.Minute
	statsInterval     = 60 * time.Minute
	maxRetries        = 3
	syncRetries       = 3
)

// pulsatrix charger implementation
type Pulsatrix struct {
	log      *util.Logger
	mu       sync.RWMutex
	conn     *websocket.Conn
	hostname string
	uri      string
	enabled  int32 // atomic for thread-safe access
	data     *util.Monitor[pulsatrixData]
	stats    *connectionStats
	cancel   context.CancelFunc // for graceful shutdown
	wg       sync.WaitGroup     // for goroutine synchronization
}

type connectionStats struct {
	mu                         sync.RWMutex
	connects                   int64 // atomic counters
	disconnects                int64
	reconnects                 int64
	writeErrors                int64
	readErrors                 int64
	heartbeatErrors            int64
	consecutiveReadErrors      int32
	consecutiveHeartbeatErrors int32
	lastConnect                time.Time
	lastError                  time.Time
	totalUptime                time.Duration
	currentUptime              time.Time
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

// NewPulsatrixFromConfig creates a pulsatrix charger from generic config
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
	// check sponsor authorization early (fail fast)
	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	wb := &Pulsatrix{
		log:      util.NewLogger("pulsatrix"),
		hostname: hostname,
		uri:      fmt.Sprintf("ws://%s/api/ws", hostname),
		data:     util.NewMonitor[pulsatrixData](dataTimeout),
		stats:    &connectionStats{},
	}

	if err := wb.connectWs(); err != nil {
		return nil, fmt.Errorf("initial connection failed: %w", err)
	}

	return wb, nil
}

// connectWs connects to a pulsatrix SECC via websocket
func (c *Pulsatrix) connectWs() error {
	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

	c.log.DEBUG.Printf("connecting to pulsatrix SECC at %s", c.hostname)

	conn, _, err := websocket.Dial(ctx, c.uri, &websocket.DialOptions{
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		return fmt.Errorf("websocket dial to pulsatrix SECC at %s failed: %w", c.hostname, err)
	}

	c.mu.Lock()
	// close existing connection if present
	if c.conn != nil {
		c.conn.Close(websocket.StatusNormalClosure, "replacing connection")
	}
	c.conn = conn
	c.stats.recordConnect()

	// create context for shutdown handling
	ctx, c.cancel = context.WithCancel(context.Background())
	c.mu.Unlock()

	// sync with retry mechanism
	if err := c.syncWithRetry(); err != nil {
		conn.Close(websocket.StatusInternalError, "sync failed")
		return fmt.Errorf("sync failed: %w", err)
	}

	// start background routines
	c.wg.Add(2)
	go c.wsReader(ctx)
	go c.heartbeat(ctx)

	c.log.INFO.Printf("connected to pulsatrix SECC at %s", c.hostname)
	return nil
}

// syncWithRetry attempts synchronization with retry mechanism
func (c *Pulsatrix) syncWithRetry() error {
	for i := 0; i < syncRetries; i++ {
		if err := c.Enable(false); err == nil {
			return nil
		}
		if i < syncRetries-1 {
			time.Sleep(time.Second)
		}
	}
	return fmt.Errorf("sync with pulsatrix SECC at %s failed after %d attempts", c.hostname, syncRetries)
}

// reconnectWs reconnects to a pulsatrix SECC websocket
func (c *Pulsatrix) reconnectWs() {
	c.stats.recordReconnect()

	bo := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(2*time.Second),
		backoff.WithMaxInterval(30*time.Second),
		backoff.WithMultiplier(1.5),
		backoff.WithMaxElapsedTime(15*time.Minute),
	)

	operation := func() error {
		return c.connectWs()
	}

	if err := backoff.Retry(operation, bo); err != nil {
		c.log.ERROR.Printf("reconnect to pulsatrix SECC at %s failed after 15min: %v", c.hostname, err)
	}
}

// wsReader runs a loop that reads messages from the websocket
func (c *Pulsatrix) wsReader(ctx context.Context) {
	defer c.wg.Done()
	defer func() {
		c.mu.Lock()
		if c.conn != nil {
			c.stats.recordDisconnect()
			c.conn.Close(websocket.StatusNormalClosure, "websocket reader shutting down")
			c.conn = nil
		}
		c.mu.Unlock()

		// only reconnect if not explicitly stopped
		select {
		case <-ctx.Done():
			return // shutdown requested
		default:
			time.Sleep(time.Second)
			go c.reconnectWs()
		}
	}()

	for {
		// check for context cancellation
		select {
		case <-ctx.Done():
			return
		default:
		}

		readCtx, cancel := context.WithTimeout(ctx, request.Timeout)
		messageType, message, err := c.getConn().Read(readCtx)
		cancel()

		if err != nil {
			// check if context was cancelled (graceful shutdown)
			if ctx.Err() != nil {
				return
			}

			c.stats.recordReadError()
			// warn only after consecutive errors
			if c.stats.shouldWarnRead() {
				c.log.WARN.Printf("websocket read on pulsatrix SECC at %s failed %d times consecutively: %v",
					c.hostname, atomic.LoadInt32(&c.stats.consecutiveReadErrors), err)
			} else {
				c.log.TRACE.Printf("websocket read error on pulsatrix SECC at %s (attempt %d of %d): %v",
					c.hostname, atomic.LoadInt32(&c.stats.consecutiveReadErrors), maxRetries, err)
			}
			return // trigger defer reconnect
		}

		// reset error counter after successful read
		c.stats.resetReadErrors()
		c.parseWsMessage(messageType, message)
	}
}

// getConn returns the current connection in a thread-safe manner
func (c *Pulsatrix) getConn() *websocket.Conn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

// write writes a message to the websocket
func (c *Pulsatrix) write(message string) error {
	conn := c.getConn()
	if conn == nil {
		return fmt.Errorf("websocket not connected to pulsatrix SECC at %s", c.hostname)
	}

	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

	if err := conn.Write(ctx, websocket.MessageText, []byte(message)); err != nil {
		c.stats.recordWriteError()
		c.log.WARN.Printf("write to pulsatrix SECC at %s failed: %v - trying reconnect", c.hostname, err)
		go c.reconnectWs() // async reconnect
		return err
	}
	return nil
}

// parseWsMessage parses a message from the websocket
func (c *Pulsatrix) parseWsMessage(messageType websocket.MessageType, message []byte) {
	if messageType != websocket.MessageText {
		return
	}

	if bytes.Contains(message, []byte(":NaN")) {
		message = bytes.ReplaceAll(message, []byte(":NaN"), []byte(":null"))
	}

	var parsedMessage struct {
		Message json.RawMessage `json:"message"`
	}

	if err := json.Unmarshal(message, &parsedMessage); err != nil {
		c.log.DEBUG.Printf("failed to unmarshal websocket message: %v", err)
		return
	}

	val, _ := c.data.Get()
	if err := json.Unmarshal(parsedMessage.Message, &val); err != nil {
		c.log.DEBUG.Printf("failed to unmarshal message content: %v", err)
	} else {
		c.data.Set(val)
	}
}

// heartbeat sends a heartbeat to the pulsatrix SECC
func (c *Pulsatrix) heartbeat(ctx context.Context) {
	defer c.wg.Done()

	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	statsTicker := time.NewTicker(statsInterval)
	defer statsTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeatTicker.C:
			enabled := atomic.LoadInt32(&c.enabled) != 0
			if err := c.Enable(enabled); err != nil {
				c.stats.recordHeartbeatError()

				// warn only after consecutive failures
				if c.stats.shouldWarnHeartbeat() {
					c.log.WARN.Printf("heartbeat with pulsatrix SECC at %s failed %d times consecutively: %v",
						c.hostname, atomic.LoadInt32(&c.stats.consecutiveHeartbeatErrors), err)
				} else {
					c.log.TRACE.Printf("heartbeat failure on pulsatrix SECC at %s (attempt %d of %d): %v",
						c.hostname, atomic.LoadInt32(&c.stats.consecutiveHeartbeatErrors), maxRetries, err)
				}
			} else {
				// reset error counter after successful heartbeat
				c.stats.resetHeartbeatErrors()
			}
		case <-statsTicker.C:
			c.logConnectionStats()
		}
	}
}

// Shutdown gracefully closes the connection and stops all goroutines
func (c *Pulsatrix) Shutdown() error {
	if c.cancel != nil {
		c.cancel()
	}

	// wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("shutdown timeout")
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
	enabled := atomic.LoadInt32(&c.enabled) != 0
	return verifyEnabled(c, enabled)
}

// Enable implements the api.Charger interface
func (c *Pulsatrix) Enable(enable bool) error {
	message := fmt.Sprintf("setEnabled\n%t", enable)
	if err := c.write(message); err != nil {
		return err
	}

	var enabledVal int32
	if enable {
		enabledVal = 1
	}
	atomic.StoreInt32(&c.enabled, enabledVal)
	return nil
}

// MaxCurrent implements the api.CurrentLimiter interface
func (c *Pulsatrix) MaxCurrent(current int64) error {
	return c.MaxCurrentMillis(float64(current))
}

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *Pulsatrix) MaxCurrentMillis(current float64) error {
	message := fmt.Sprintf("setCurrentLimit\n%g", current)
	return c.write(message)
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

// Phases1p3p implements the api.PhaseSwitcher interface
func (c *Pulsatrix) Phases1p3p(phases int) error {
	message := fmt.Sprintf("set1p3p\n%t", phases == 1)
	return c.write(message)
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

// connection statistics methods

func (s *connectionStats) recordConnect() {
	atomic.AddInt64(&s.connects, 1)
	s.mu.Lock()
	s.lastConnect = time.Now()
	s.currentUptime = time.Now()
	s.mu.Unlock()
	atomic.StoreInt32(&s.consecutiveReadErrors, 0)
	atomic.StoreInt32(&s.consecutiveHeartbeatErrors, 0)
}

func (s *connectionStats) recordDisconnect() {
	atomic.AddInt64(&s.disconnects, 1)
	s.mu.Lock()
	if !s.currentUptime.IsZero() {
		s.totalUptime += time.Since(s.currentUptime)
		s.currentUptime = time.Time{}
	}
	s.mu.Unlock()
}

func (s *connectionStats) recordReconnect() {
	atomic.AddInt64(&s.reconnects, 1)
}

func (s *connectionStats) recordWriteError() {
	atomic.AddInt64(&s.writeErrors, 1)
	s.mu.Lock()
	s.lastError = time.Now()
	s.mu.Unlock()
}

func (s *connectionStats) recordReadError() {
	atomic.AddInt64(&s.readErrors, 1)
	atomic.AddInt32(&s.consecutiveReadErrors, 1)
	s.mu.Lock()
	s.lastError = time.Now()
	s.mu.Unlock()
}

func (s *connectionStats) recordHeartbeatError() {
	atomic.AddInt64(&s.heartbeatErrors, 1)
	atomic.AddInt32(&s.consecutiveHeartbeatErrors, 1)
	s.mu.Lock()
	s.lastError = time.Now()
	s.mu.Unlock()
}

func (s *connectionStats) resetReadErrors() {
	atomic.StoreInt32(&s.consecutiveReadErrors, 0)
}

func (s *connectionStats) resetHeartbeatErrors() {
	atomic.StoreInt32(&s.consecutiveHeartbeatErrors, 0)
}

func (s *connectionStats) shouldWarnRead() bool {
	return atomic.LoadInt32(&s.consecutiveReadErrors) >= maxRetries
}

func (s *connectionStats) shouldWarnHeartbeat() bool {
	return atomic.LoadInt32(&s.consecutiveHeartbeatErrors) >= maxRetries
}

func (c *Pulsatrix) logConnectionStats() {
	s := c.stats

	// atomic reads - no lock needed for counters
	connects := atomic.LoadInt64(&s.connects)
	disconnects := atomic.LoadInt64(&s.disconnects)
	reconnects := atomic.LoadInt64(&s.reconnects)
	writeErrors := atomic.LoadInt64(&s.writeErrors)
	readErrors := atomic.LoadInt64(&s.readErrors)
	heartbeatErrors := atomic.LoadInt64(&s.heartbeatErrors)

	// only lock for time-related data
	s.mu.RLock()
	uptime := s.totalUptime
	if !s.currentUptime.IsZero() {
		uptime += time.Since(s.currentUptime)
	}
	lastError := s.lastError
	s.mu.RUnlock()

	c.log.DEBUG.Printf("pulsatrix SECC at %s connection stats: connects: %d, disconnects: %d, reconnects: %d",
		c.hostname, connects, disconnects, reconnects)
	c.log.DEBUG.Printf("pulsatrix SECC at %s error stats: writeErrors: %d, readErrors: %d, heartbeatErrors: %d, uptime: %v",
		c.hostname, writeErrors, readErrors, heartbeatErrors, uptime)

	if !lastError.IsZero() {
		c.log.DEBUG.Printf("pulsatrix SECC at %s stats: last error: %v ago", c.hostname, time.Since(lastError))
	}
}
