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

	ct.StartCharge()
	clck.Add(time.Hour)
	ct.StopCharge()
	clck.Add(time.Hour)

	d, err := ct.ChargingTime()

	if d != time.Hour || err != nil {
		t.Error(d, err)
	}
}
