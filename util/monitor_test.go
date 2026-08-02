package util

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
)

func TestMonitorRacyMaps(t *testing.T) {
	m := NewMonitor[map[int]int](0)
	done := make(chan struct{})

	time.AfterFunc(time.Second, func() { close(done) })

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				m.SetFunc(func(mm map[int]int) map[int]int {
					if mm == nil {
						mm = make(map[int]int)
					}
					i := rand.Int()
					mm[i] = i
					return mm
				})
			}
		}
	}()

	<-m.Done()

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				err := m.GetFunc(func(mm map[int]int) {
					for k, v := range mm {
						assert.Equal(t, k, v)
					}
				})
				assert.NoError(t, err)
			}
		}
	}()

	<-done
}

func TestMonitorWithoutTimeout(t *testing.T) {
	clock := clock.NewMock()
	m := NewMonitor[int](0).WithClock(clock)

	_, err := m.Get()
	assert.Error(t, err)

	m.Set(0)
	_, err = m.Get()
	assert.NoError(t, err)

	clock.Add(time.Hour)
	_, err = m.Get()
	assert.NoError(t, err)
}

func TestMonitorGetContext(t *testing.T) {
	// a long timeout would block the first-read wait; a cancelled context cuts it short
	m := NewMonitor[int](time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := m.GetContext(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), time.Second, "context must cancel the first-read wait")
}
