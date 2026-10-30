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
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// fakePublisher records every message instead of talking to a real broker.
type fakePublisher struct {
	mu   sync.Mutex
	msgs []fakeMsg
}

type fakeMsg struct {
	topic    string
	retained bool
	payload  string
}

func (f *fakePublisher) Publish(topic string, retained bool, payload string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.msgs = append(f.msgs, fakeMsg{topic, retained, payload})
}

// last returns the most recent message published on the given sub-topic. wb's
// own publish() always prepends wb.base ("device/evCharger/0/..."), so this
// matches on the suffix rather than the bare name callers pass here.
func (f *fakePublisher) last(subTopic string) (fakeMsg, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := len(f.msgs) - 1; i >= 0; i-- {
		if strings.HasSuffix(f.msgs[i].topic, "/"+subTopic) {
			return f.msgs[i], true
		}
	}
	return fakeMsg{}, false
}

// newTestAmbibox builds a bare Ambibox with every getter stubbed to a benign
// default (live, connected, EV connected, idle, nothing blocked) and a
// fakePublisher in place of a real MQTT client.
func newTestAmbibox() (*Ambibox, *fakePublisher) {
	pub := &fakePublisher{}
	wb := &Ambibox{
		log:      util.NewLogger("test"),
		client:   pub,
		base:     "device/evCharger/0",
		minPower: 600,
		phases:   3,

		liveG:         func() (float64, error) { return 50, nil },
		connectedG:    func() (bool, error) { return true, nil },
		evConnectedG:  func() (bool, error) { return true, nil },
		replugG:       func() (bool, error) { return false, nil },
		wakeBlockedG:  func() (bool, error) { return false, nil },
		sessionStateG: func() (string, error) { return ambiStateStopped, nil },
		controlModeG:  func() (string, error) { return "LIMITABLE", nil },
		powerG:        func() (float64, error) { return 0, nil },
		chargeMinG:    func() (float64, error) { return 0, api.ErrNotAvailable },
		chargeMaxG:    func() (float64, error) { return 10500, nil },
	}
	for i := range wb.voltG {
		wb.voltG[i] = func() (float64, error) { return 230, nil }
	}
	for i := range wb.currG {
		wb.currG[i] = func() (float64, error) { return 0, nil }
	}
	return wb, pub
}

func TestAmbiboxChargeWatts(t *testing.T) {
	wb, _ := newTestAmbibox()

	if p := wb.chargeWatts(0); p != 0 {
		t.Errorf("chargeWatts(0) = %v, want 0", p)
	}

	// 6A * 3 phases * 230V = 4140W, above the minimum - passes through.
	if p := wb.chargeWatts(6); p != 4140 {
		t.Errorf("chargeWatts(6) = %v, want 4140", p)
	}

	// A tiny current must not throttle the session into an alarm - it asks
	// for the box's minimum instead.
	if p := wb.chargeWatts(0.5); p != wb.minPower {
		t.Errorf("chargeWatts(0.5) = %v, want the minimum %v", p, wb.minPower)
	}

	// Clamped at the box's own reported maximum.
	wb.chargeMaxG = func() (float64, error) { return 5000, nil }
	if p := wb.chargeWatts(32); p != 5000 {
		t.Errorf("chargeWatts(32) with a 5000W ceiling = %v, want 5000", p)
	}
}

func TestAmbiboxChargeWattsReportedMinimum(t *testing.T) {
	wb, _ := newTestAmbibox()

	wb.chargeMinG = func() (float64, error) { return 1000, nil }
	if p := wb.chargeWatts(0.5); p != 1000 {
		t.Errorf("chargeWatts(0.5) with chargePowerMin=1000 = %v, want 1000", p)
	}

	// a reported minimum below the configured one does not lower it
	wb.chargeMinG = func() (float64, error) { return 400, nil }
	if p := wb.chargeWatts(0.5); p != wb.minPower {
		t.Errorf("chargeWatts(0.5) with chargePowerMin=400 = %v, want %v", p, wb.minPower)
	}
}

func TestAmbiboxTryWake(t *testing.T) {
	t.Run("idle - wakes", func(t *testing.T) {
		wb, pub := newTestAmbibox()
		wb.tryWake()
		if msg, ok := pub.last("wakeUp"); !ok || msg.payload != "true" {
			t.Errorf("published %+v, want wakeUp=true", msg)
		}
	})

	t.Run("replugRequired - does not wake", func(t *testing.T) {
		wb, pub := newTestAmbibox()
		wb.replugG = func() (bool, error) { return true, nil }
		wb.tryWake()
		if _, ok := pub.last("wakeUp"); ok {
			t.Error("wakeUp must not be sent while replugRequired")
		}
	})

	t.Run("wakeUpBlocked - does not wake", func(t *testing.T) {
		wb, pub := newTestAmbibox()
		wb.wakeBlockedG = func() (bool, error) { return true, nil }
		wb.tryWake()
		if _, ok := pub.last("wakeUp"); ok {
			t.Error("wakeUp must not be sent while wakeUpBlocked")
		}
	})
}

func TestAmbiboxStatus(t *testing.T) {
	t.Run("box unreachable - StatusNone/ErrTimeout", func(t *testing.T) {
		wb, _ := newTestAmbibox()
		wb.liveG = func() (float64, error) { return 0, errors.New("timeout") }
		if s, err := wb.Status(); !errors.Is(err, api.ErrTimeout) || s != api.StatusNone {
			t.Errorf("Status() = %v, %v, want StatusNone, ErrTimeout", s, err)
		}
	})

	t.Run("no cable - StatusA", func(t *testing.T) {
		wb, _ := newTestAmbibox()
		wb.evConnectedG = func() (bool, error) { return false, nil }
		s, err := wb.Status()
		if err != nil || s != api.StatusA {
			t.Errorf("Status() = %v, %v, want StatusA, nil", s, err)
		}
	})

	t.Run("connected, idle - StatusB", func(t *testing.T) {
		wb, _ := newTestAmbibox()
		s, err := wb.Status()
		if err != nil || s != api.StatusB {
			t.Errorf("Status() = %v, %v, want StatusB, nil", s, err)
		}
	})

	t.Run("charging - StatusC", func(t *testing.T) {
		wb, _ := newTestAmbibox()
		wb.sessionStateG = func() (string, error) { return ambiStateChargeLoop, nil }
		s, err := wb.Status()
		if err != nil || s != api.StatusC {
			t.Errorf("Status() = %v, %v, want StatusC, nil", s, err)
		}
	})
}

func TestAmbiboxStatusReason(t *testing.T) {
	t.Run("replug required", func(t *testing.T) {
		wb, _ := newTestAmbibox()
		wb.replugG = func() (bool, error) { return true, nil }
		if r, _ := wb.StatusReason(); r != api.ReasonDisconnectRequired {
			t.Errorf("StatusReason() = %v, want ReasonDisconnectRequired", r)
		}
	})

	t.Run("authorization pending", func(t *testing.T) {
		wb, _ := newTestAmbibox()
		wb.sessionStateG = func() (string, error) { return ambiStateAuthorization, nil }
		if r, _ := wb.StatusReason(); r != api.ReasonWaitingForAuthorization {
			t.Errorf("StatusReason() = %v, want ReasonWaitingForAuthorization", r)
		}
	})
}

func TestAmbiboxEnabledAdoptsRunningSession(t *testing.T) {
	wb, _ := newTestAmbibox()
	wb.sessionStateG = func() (string, error) { return ambiStateChargeLoop, nil }
	enabled, err := wb.Enabled()
	if err != nil || !enabled {
		t.Fatalf("Enabled() = %v, %v, want true, nil (adopt a running session on first read)", enabled, err)
	}
}

func TestAmbiboxEnabledPropagatesSessionStateErrorWithoutCaching(t *testing.T) {
	wb, _ := newTestAmbibox()
	wantErr := errors.New("sessionState not delivered yet")
	wb.sessionStateG = func() (string, error) { return "", wantErr }

	if _, err := wb.Enabled(); !errors.Is(err, wantErr) {
		t.Fatalf("Enabled() err = %v, want %v (not silently adopted as disabled)", err, wantErr)
	}
	if wb.enabled != nil {
		t.Fatalf("Enabled() cached %v after a getter error, want no adoption so the next read retries", *wb.enabled)
	}

	// once the retained message arrives, adoption succeeds normally
	wb.sessionStateG = func() (string, error) { return ambiStateChargeLoop, nil }
	if enabled, err := wb.Enabled(); err != nil || !enabled {
		t.Fatalf("Enabled() after recovery = %v, %v, want true, nil", enabled, err)
	}
}

func TestAmbiboxEnabledWithoutVehicleIsNotAnError(t *testing.T) {
	// Without a vehicle the box never publishes sessionState at all - a
	// permanent, not transient, getter error. Must read as "no session",
	// not get propagated as if the retained message were merely delayed.
	wb, _ := newTestAmbibox()
	wb.evConnectedG = func() (bool, error) { return false, nil }
	wb.sessionStateG = func() (string, error) { return "", errors.New("no retained message") }

	enabled, err := wb.Enabled()
	if err != nil || enabled {
		t.Fatalf("Enabled() = %v, %v, want false, nil (no vehicle, not a race)", enabled, err)
	}
}

func TestAmbiboxStatusPropagatesSessionStateError(t *testing.T) {
	wb, _ := newTestAmbibox()
	wantErr := errors.New("sessionState not delivered yet")
	wb.sessionStateG = func() (string, error) { return "", wantErr }

	if _, err := wb.Status(); !errors.Is(err, wantErr) {
		t.Fatalf("Status() err = %v, want %v (not silently reported as StatusB)", err, wantErr)
	}
}

func TestAmbiboxCapacityRequiresLiveness(t *testing.T) {
	wb, _ := newTestAmbibox()
	wb.capacityG = func() (float64, error) { return 50000, nil }
	wb.liveG = func() (float64, error) { return 0, errors.New("timeout") }

	if _, err := wb.Capacity(); !errors.Is(err, api.ErrTimeout) {
		t.Errorf("Capacity() err = %v, want ErrTimeout (an offline box must not report its last cached capacity)", err)
	}
}

func TestAmbiboxChargedEnergy(t *testing.T) {
	t.Run("no session - suppresses the missing-value error", func(t *testing.T) {
		wb, _ := newTestAmbibox()
		wb.energySessImpG = func() (float64, error) { return 0, api.ErrNotAvailable }
		if f, err := wb.ChargedEnergy(); err != nil || f != 0 {
			t.Errorf("ChargedEnergy() = %v, %v, want 0, nil between sessions", f, err)
		}
	})

	t.Run("active session - propagates a real getter error", func(t *testing.T) {
		wb, _ := newTestAmbibox()
		wb.sessionStateG = func() (string, error) { return ambiStateChargeLoop, nil }
		wantErr := errors.New("broker read failed")
		wb.energySessImpG = func() (float64, error) { return 0, wantErr }
		if _, err := wb.ChargedEnergy(); !errors.Is(err, wantErr) {
			t.Errorf("ChargedEnergy() err = %v, want %v (must not report a failed reading as a valid zero)", err, wantErr)
		}
	})
}

func TestAmbiboxEnable(t *testing.T) {
	t.Run("true - publishes limit, clears stopCharge, wakes", func(t *testing.T) {
		wb, pub := newTestAmbibox()
		wb.current = 6

		if err := wb.Enable(true); err != nil {
			t.Fatalf("Enable(true) = %v, want nil", err)
		}
		if msg, ok := pub.last("limitChargePower"); !ok || msg.payload != "4140" {
			t.Errorf("published %+v, want limitChargePower=4140", msg)
		}
		if msg, ok := pub.last("stopCharge"); !ok || msg.payload != "false" {
			t.Errorf("published %+v, want stopCharge=false", msg)
		}
		if msg, ok := pub.last("wakeUp"); !ok || msg.payload != "true" {
			t.Errorf("published %+v, want wakeUp=true", msg)
		}
		if enabled, err := wb.Enabled(); err != nil || !enabled {
			t.Errorf("Enabled() = %v, %v, want true, nil", enabled, err)
		}
	})

	t.Run("false - sends stopCharge", func(t *testing.T) {
		wb, pub := newTestAmbibox()

		if err := wb.Enable(false); err != nil {
			t.Fatalf("Enable(false) = %v, want nil", err)
		}
		if msg, ok := pub.last("stopCharge"); !ok || msg.payload != "true" {
			t.Errorf("published %+v, want stopCharge=true", msg)
		}
		if enabled, err := wb.Enabled(); err != nil || enabled {
			t.Errorf("Enabled() = %v, %v, want false, nil", enabled, err)
		}
	})
}

func TestAmbiboxMaxCurrentMillisOnlyPublishesWhileEnabled(t *testing.T) {
	wb, pub := newTestAmbibox()

	if err := wb.MaxCurrentMillis(6); err != nil {
		t.Fatalf("MaxCurrentMillis() = %v, want nil", err)
	}
	if _, ok := pub.last("limitChargePower"); ok {
		t.Error("limitChargePower must not be published while disabled")
	}

	_ = wb.Enable(true)
	if err := wb.MaxCurrentMillis(10); err != nil {
		t.Fatalf("MaxCurrentMillis() = %v, want nil", err)
	}
	if msg, ok := pub.last("limitChargePower"); !ok || msg.payload != "6900" {
		t.Errorf("published %+v, want limitChargePower=6900", msg)
	}
}
