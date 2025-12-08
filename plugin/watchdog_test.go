package plugin

import (
	"errors"
	"math/rand/v2"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestWatchdogSetterConcurrency(t *testing.T) {
	p := &watchdogPlugin{
		log:     util.NewLogger("foo"),
		timeout: 10 * time.Nanosecond,
	}

	var u atomic.Uint32

	set := setter(p, func(i int) error {
		if !u.CompareAndSwap(0, 1) {
			return errors.New("race")
		}

		time.Sleep(time.Duration(rand.Int32N(int32(p.timeout))))

		if !u.CompareAndSwap(1, 0) {
			return errors.New("race")
		}

		return nil
	}, nil)

	var eg errgroup.Group

	for range 100 {
		eg.Go(func() error {
			return set(1)
		})
	}

	require.NoError(t, eg.Wait())
}
