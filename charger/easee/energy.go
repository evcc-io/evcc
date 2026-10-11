package easee

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
	"math"
	"time"
)

// EnergyReading is an energy value with its observation timestamp.
type EnergyReading struct {
	Value float64
	Time  time.Time
}

// EnergyMeter derives an absolute meter from session and lifetime observations.
type EnergyMeter struct {
	Start           *ChargingSessionStartData
	Stop            *ChargingSessionData
	StopTime        time.Time
	Session         EnergyReading
	Lifetime        EnergyReading
	ConnectionStart time.Time
	Initialized     bool
}

// ValidEnergy reports whether a value is finite and nonnegative.
func ValidEnergy(value float64) bool {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return false
	}
	return value >= 0
}

func (m *EnergyMeter) sessionMatched() bool {
	if m.Start == nil || m.Start.Start.IsZero() {
		return false
	}
	if m.Start.Start.Before(m.ConnectionStart) || m.Session.Time.Before(m.Start.Start) {
		return false
	}
	return ValidEnergy(m.Start.MeterValue + m.Session.Value)
}

// Completed reports whether the current session has a completion record.
func (m *EnergyMeter) Completed() bool {
	return m.Stop != nil && (m.Start == nil || m.Stop.ID == m.Start.ID)
}

func (m *EnergyMeter) activeSession() bool {
	if m.Completed() {
		return false
	}
	return m.sessionMatched()
}

func (m *EnergyMeter) completedSince(since time.Time) bool {
	if m.Stop == nil || m.Stop.Stop.IsZero() || m.StopTime.Before(since) {
		return false
	}
	return m.Completed()
}

// Total returns the highest absolute reading supported by the current observations.
func (m *EnergyMeter) Total() float64 {
	total := m.Lifetime.Value
	if m.Stop != nil {
		total = max(total, m.Stop.MeterValueStop)
	}
	if m.Start != nil {
		total = max(total, m.Start.MeterValue)
		if m.activeSession() {
			total = max(total, m.Start.MeterValue+m.Session.Value)
		}
	}
	return total
}

// Ready reports whether the observations establish an energy meter for the operation mode.
func (m *EnergyMeter) Ready(mode int, modeTime time.Time) bool {
	if modeTime.IsZero() {
		return false
	}
	if mode == ModeCharging {
		return m.activeSession()
	}
	if m.completedSince(modeTime) {
		return true
	}
	if mode >= ModeAwaitingStart && m.activeSession() {
		return true
	}
	// An hourly reading predating the last state change cannot repair history safely.
	if m.Lifetime.Time.IsZero() || m.Lifetime.Time.Before(modeTime) || m.Lifetime.Time.Before(m.Session.Time) {
		return false
	}
	if m.Start != nil && m.Lifetime.Time.Before(m.Start.Start) {
		return false
	}
	return true
}
