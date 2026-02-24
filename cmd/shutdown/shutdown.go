package shutdown

import (
	"sync"
)

var (
	mu       sync.Mutex
	handlers = make([]func(), 0)
)

// Register registers a function for executing on application shutdown
func Register(cb func()) {
	mu.Lock()
	handlers = append(handlers, cb)
	mu.Unlock()
}

// Cleanup executes the registered shutdown functions when the stop channel closes
func Cleanup(doneC chan struct{}) {
	var wg sync.WaitGroup

	mu.Lock()
	for _, cb := range handlers {
		wg.Go(cb)
	}
	mu.Unlock()

	wg.Wait()
	close(doneC)
}
