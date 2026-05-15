package core

import (
	"testing"
)

func (lp *Loadpoint) RLock() {
	if testing.Testing() && lp.rwMutex.Add(1) > 1 {
		panic("reentrant RLock")
	}
	lp.RWMutex.RLock()
}

func (lp *Loadpoint) RUnlock() {
	if testing.Testing() {
		lp.rwMutex.Add(-1)
	}
	lp.RWMutex.RUnlock()
}

func (lp *Loadpoint) Lock() {
	if testing.Testing() && lp.rwMutex.Add(1) > 1 {
		panic("reentrant Lock")
	}
	lp.RWMutex.Lock()
}

func (lp *Loadpoint) Unlock() {
	if testing.Testing() {
		lp.rwMutex.Add(-1)
	}
	lp.RWMutex.Unlock()
}
