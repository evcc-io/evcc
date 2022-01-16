package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
)

func TestTimer(t *testing.T) {
	at := NewActiveTimer()
	clck := clock.NewMock()
	at.clck = clck

	at.Start()
	clck.Add(time.Minute)
	at.Reset()
	clck.Add(time.Minute)

	if d := int(at.duration().Seconds()); d != int(time.Minute.Seconds()) {
		t.Error(d)
	}

	// continue
	at.Start()
	clck.Add(2 * time.Minute)
	at.Stop()

	if d := int(at.duration().Seconds()); d != int(2*time.Minute.Seconds()) {
		t.Error(d)
	}
	// continue
	at.Start()
	clck.Add(1 * time.Minute)
	at.Stop()

	if d := int(at.duration().Seconds()); d != int(time.Minute.Seconds()) {
		t.Error(d)
	}

}
