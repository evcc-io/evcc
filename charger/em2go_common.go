package charger

// LICENSE

// Copyright (c) evcc.io (andig, naltatis, premultiply)

// This module is NOT covered by the MIT license. All rights reserved.

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// newCommsBackoff returns the shared exponential backoff configuration used
// during em2go charger initialization to handle transient TCP timeouts.
func newCommsBackoff(ctx context.Context) backoff.BackOffContext {
	return backoff.WithContext(
		backoff.NewExponentialBackOff(
			backoff.WithInitialInterval(2*time.Second),
			backoff.WithMaxInterval(10*time.Second),
			backoff.WithMaxElapsedTime(30*time.Second),
		),
		ctx,
	)
}

// heartbeat keeps the Modbus connection alive to prevent the charger from
// entering its failsafe state when the configured communication timeout expires.
func heartbeat(ctx context.Context, log *util.Logger, conn *modbus.Connection, reg uint16, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
		if _, err := conn.ReadHoldingRegisters(reg, 1); err != nil {
			log.ERROR.Println("heartbeat:", err)
		}
	}
}

// setupFailsafeHeartbeat reads the charger's communication timeout register and,
// if non-zero, starts a heartbeat goroutine that pings the charger at half the
// timeout interval to prevent the failsafe from triggering during normal operation.
// The backoff retry allows the charger's TCP stack time to come up after a reboot.
func setupFailsafeHeartbeat(ctx context.Context, log *util.Logger, conn *modbus.Connection, timeoutReg, heartbeatReg uint16) error {
	var timeout uint16
	if err := backoff.RetryNotify(func() error {
		b, err := conn.ReadHoldingRegisters(timeoutReg, 1)
		if err != nil {
			return err
		}
		timeout = binary.BigEndian.Uint16(b)
		return nil
	}, newCommsBackoff(ctx), func(err error, d time.Duration) {
		log.WARN.Printf("charger not reachable, retrying in %v: %v", d, err)
	}); err != nil {
		return fmt.Errorf("failsafe timeout: %w", err)
	}

	if timeout > 0 {
		interval := time.Duration(timeout) * time.Second / 2
		log.DEBUG.Printf("failsafe timeout: %ds, heartbeat interval: %v", timeout, interval)
		go heartbeat(ctx, log, conn, heartbeatReg, interval)
	}

	return nil
}
