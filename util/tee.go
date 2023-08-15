package util

import (
	"reflect"
	"sync"
)

// TeeAttacher allows attaching a listener to a tee
type TeeAttacher interface {
	Attach() <-chan Param
}

// Tee distributes parameters to subscribers
type Tee struct {
	recv []chan<- Param
	mu   sync.Mutex // Mutex to protect recv slice
}

// Attach creates a new receiver channel and attaches it to the tee
func (t *Tee) Attach() <-chan Param {
	out := make(chan Param, 16) // Use a buffered channel to prevent blocking
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
		// Convert the parameter once instead of per receiver iteration
		if val := reflect.ValueOf(msg.Val); val.Kind() == reflect.Ptr {
			if ptr := reflect.Indirect(val); ptr.IsValid() {
				msg.Val = ptr.Addr().Elem().Interface()
			}
		}

		t.mu.Lock()
		for _, recv := range t.recv {
			recv <- msg
		}
		t.mu.Unlock()
	}
}
