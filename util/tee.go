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
	recv     []chan<- Param
	attachCh chan chan<- Param // Channel for attaching new receivers
	detachCh chan chan<- Param // Channel for detaching receivers
	wg       sync.WaitGroup     // WaitGroup for graceful shutdown
	lock     sync.Mutex        // Mutex for safe concurrent access
}

// Attach creates a new receiver channel and attaches it to the tee
func (t *Tee) Attach() <-chan Param {
	out := make(chan Param, 16)
	t.attachCh <- out
	return out
}

// Run starts parameter distribution
func (t *Tee) Run(in <-chan Param) {
	for {
		select {
		case recv := <-t.attachCh:
			t.lock.Lock()
			t.add(recv)
			t.lock.Unlock()
		case recv := <-t.detachCh:
			t.lock.Lock()
			t.remove(recv)
			t.lock.Unlock()
		case msg, ok := <-in:
			if !ok {
				// Input channel closed, terminate distribution
				t.lock.Lock()
				for _, recv := range t.recv {
					close(recv)
				}
				t.lock.Unlock()
				t.wg.Done()
				return
			}

			t.lock.Lock()
			for _, recv := range t.recv {
				// dereference pointers
				if val := reflect.ValueOf(msg.Val); val.Kind() == reflect.Ptr {
					if ptr := reflect.Indirect(val); ptr.IsValid() {
						msg.Val = ptr.Interface()
					}
				}

				recv <- msg
			}
			t.lock.Unlock()
		}
	}
}

func (t *Tee) add(out chan<- Param) {
	t.recv = append(t.recv, out)
}

func (t *Tee) remove(out chan<- Param) {
	for i, recv := range t.recv {
		if recv == out {
			close(recv)
			t.recv = append(t.recv[:i], t.recv[i+1:]...)
			break
		}
	}
}

// Param represents a parameter
type Param struct {
	Val interface{}
}
