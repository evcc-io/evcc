package wrapper

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
)

func TestTimer(t *testing.T) {
	ct := NewChargeTimer()
	clock := clock.NewMock()
	ct.clock = clock

	ct.StartCharge(false)
	clock.Add(time.Hour)
	ct.StopCharge()
	clock.Add(time.Hour)

	if d, err := ct.ChargeDuration(); d != 1*time.Hour || err != nil {
		t.Error(d, err)
	}

	// continue
	ct.StartCharge(true)
	clock.Add(2 * time.Hour)
	ct.StopCharge()

	if d, err := ct.ChargeDuration(); d != 3*time.Hour || err != nil {
		t.Error(d, err)
	}
}
