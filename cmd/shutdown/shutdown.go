package shutdown

import (
	"sync"
	"time"
)

var (
	mu       sync.Mutex
	handlers = make([]func(), 0)
	exitC    = make(chan struct{})
)

func Register(cb func()) {
	mu.Lock()
	handlers = append(handlers, cb)
	mu.Unlock()
}

func Run(stopC <-chan struct{}) {
	mu.Lock()
	defer mu.Unlock()

	<-stopC
	wg := new(sync.WaitGroup)

	for _, cb := range handlers {
		wg.Add(1)

		go func(cb func()) {
			cb()
			wg.Done()
		}(cb)
	}

	wg.Wait()
	close(exitC)

	println("Run wg.Wait exitC")
}

func Done(timeout ...time.Duration) <-chan struct{} {
	println("Done")

	to := time.Second
	if len(timeout) == 1 {
		to = timeout[0]
	}

	select {
	case <-exitC:
		println("Done exitC closed")
		return exitC
	case <-time.After(to):
		println("Done timeout")
		exitC := make(chan struct{})
		close(exitC)
		return exitC
	}
}
