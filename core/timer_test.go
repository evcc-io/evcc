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
	clck.Add(20 * time.Second)
	require.Equal(t, at.Expired(), false)

	// wait another 20 sec to expire the timer - this will reset the timer as well
	clck.Add(20 * time.Second)
	require.Equal(t, at.Expired(), true)

	// start
	at.Start()
	clck.Add(time.Minute)
	require.Equal(t, at.Expired(), true)

}
