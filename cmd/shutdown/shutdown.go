package shutdown

import (
	"sync"
)

var (
	mu       sync.Mutex
	handlers = make([]func(), 0)
	exitC    = make(chan struct{})
)

// Register registers a function for executing on application shutdown
func Register(cb func()) {
	mu.Lock()
	handlers = append(handlers, cb)
	mu.Unlock()
}

// Run executes the registered shutdown functions when the stop channel closes
func Run(stopC <-chan struct{}) {
	<-stopC
	wg := new(sync.WaitGroup)

	mu.Lock()
	for _, cb := range handlers {
		wg.Add(1)

		go func(cb func()) {
			cb()
			wg.Done()
		}(cb)
	}
	mu.Unlock()

	wg.Wait()
	close(exitC)
}

// Done returns a readable channel that closes when all registered functions have completed
func Done() <-chan struct{} {
	return exitC
}
