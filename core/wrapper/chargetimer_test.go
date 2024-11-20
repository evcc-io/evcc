package wrapper

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
)

func TestTimer(t *testing.T) {
	ct := NewChargeTimer()
	clck := clock.NewMock()
	ct.clck = clck

	ct.StartCharge(false)
	clck.Add(time.Hour)
	ct.StopCharge()
	clck.Add(time.Hour)

	if d, err := ct.ChargeDuration(); d != 1*time.Hour || err != nil {
		t.Error(d, err)
	}

	// continue
	ct.StartCharge(true)
	clck.Add(2 * time.Hour)
	ct.StopCharge()

	if d, err := ct.ChargeDuration(); d != 3*time.Hour || err != nil {
		t.Error(d, err)
	}
}
