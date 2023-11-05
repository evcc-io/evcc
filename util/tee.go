package util

import (
	"fmt"
	"reflect"
	"sync"
)

// Tee distributes parameters to subscribers
type Tee struct {
	mu   sync.Mutex
	recv []chan<- Param
}

// Attach creates a new receiver channel and attaches it to the tee
func (t *Tee) Attach() <-chan Param {
	// TODO find better approach to prevent deadlocks
	// this will buffer the receiver channel to prevent deadlocks when consumers use mutex-protected loadpoint api
	out := make(chan Param, 16)
	t.add(out)
	return out
}

// add attaches a receiver channel to the tee
func (t *Tee) add(out chan<- Param) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.recv = append(t.recv, out)
}

// Run starts parameter distribution
func (t *Tee) Run(in <-chan Param) {
	for msg := range in {
		if val := reflect.ValueOf(msg.Val); val.Kind() == reflect.Ptr {
			if ptr := reflect.Indirect(val); ptr.IsValid() {
				msg.Val = ptr.Addr().Elem().Interface()
			}
		}

		t.mu.Lock()
		for i, recv := range t.recv {
			select {

			case recv <- msg:
			default:
				fmt.Println("tee blocked", i)
			}
		}
		t.mu.Unlock()
	}
}
