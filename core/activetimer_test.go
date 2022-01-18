package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/util"
)

func TestTimer(t *testing.T) {
	log := util.NewLogger("foo")
	at := NewActiveTimer(log)
	clck := clock.NewMock()
	at.clck = clck
	at.Start()
	clck.Add(time.Minute)
	at.Reset()
	clck.Add(time.Minute)

	if d := at.lastduration; d != 0 {
		t.Error(d)
	}

	// continue
	at.Start()
	clck.Add(2 * time.Minute)
	at.Stop()

	if d := int(at.lastduration.Seconds()); d != int(2*time.Minute.Seconds()) {
		t.Error(d)
	}
	// continue - should do nothing as the timer was started allready
	at.Start()
	clck.Add(1 * time.Minute)
	at.Stop()

	if d := int(at.lastduration.Seconds()); d != int(2*time.Minute.Seconds()) {
		t.Error(d)
	}

}
