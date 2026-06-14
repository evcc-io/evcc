package core

import (
	"bytes"
	"runtime"
	"strconv"
	"testing"
)

// goID returns the current goroutine id. Test-only helper for the reentrancy
// detector below, which must distinguish genuine reentrant locking (a deadlock)
// from the legitimate concurrent access by the sense and control loops.
func goID() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	// first line is "goroutine 123 [running]:"
	s := bytes.TrimPrefix(buf[:n], []byte("goroutine "))
	if i := bytes.IndexByte(s, ' '); i >= 0 {
		id, _ := strconv.ParseInt(string(s[:i]), 10, 64)
		return id
	}
	return 0
}

// The loadpoint RWMutex is accessed concurrently by the sense loop (writes
// measurements under Lock) and the control loop (reads them via RLock getters).
// rwMutex holds the goroutine id of the current writer, so the test-only detector
// can catch a goroutine that would deadlock itself by locking while it already
// holds the write lock - without flagging legitimate cross-goroutine concurrency
// the way a shared counter would.

func (lp *Loadpoint) RLock() {
	if testing.Testing() && lp.rwMutex.Load() == goID() {
		panic("RLock while holding Lock")
	}
	lp.RWMutex.RLock()
}

func (lp *Loadpoint) RUnlock() {
	lp.RWMutex.RUnlock()
}

func (lp *Loadpoint) Lock() {
	if testing.Testing() && lp.rwMutex.Load() == goID() {
		panic("reentrant Lock")
	}
	lp.RWMutex.Lock()
	if testing.Testing() {
		lp.rwMutex.Store(goID())
	}
}

func (lp *Loadpoint) Unlock() {
	if testing.Testing() {
		lp.rwMutex.Store(0)
	}
	lp.RWMutex.Unlock()
}
