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

	if d := at.duration(); d != int64(time.Minute.Seconds()) {
		t.Error(d)
	}

	// continue
	at.Start()
	clck.Add(2 * time.Minute)
	at.Stop()

	if d := at.duration(); d != int64(2*time.Minute.Seconds()) {
		t.Error(d)
	}
	// continue
	at.Start()
	clck.Add(1 * time.Minute)
	at.Stop()

	if d := at.duration(); d != int64(time.Minute.Seconds()) {
		t.Error(d)
	}

}
