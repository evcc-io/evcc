package util

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
						require.Equal(t, k, v)
					}
				})
				require.NoError(t, err)
			}
		}
	}()

	<-done
}
