package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/require"
)

func TestTimer(t *testing.T) {
	at := NewTimer()

	clck := clock.NewMock()
	at.clck = clck

	// start
	at.Start()

	// maximum 2 attempts
	at.wakeupAttemptsLeft = 2

	clck.Add(10 * time.Second)
	require.Equal(t, WakeUpTimerInactive, at.Elapsed())

	// wait another 20 sec to expire the timer - this will reset the timer as well
	clck.Add(wakeupTimeout + 10*time.Second)
	require.Equal(t, WakeUpTimerElapsed, at.Elapsed())

	// elapse
	clck.Add(time.Minute)
	require.Equal(t, WakeUpTimerElapsed, at.Elapsed())

	// elapse
	clck.Add(time.Minute)
	require.Equal(t, WakeUpTimerFinished, at.Elapsed())
}
