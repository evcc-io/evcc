package modbus

import "sync"

var mu2 sync.Mutex

func Lock() {
	mu2.Lock()
}

func Unlock() {
	mu2.Unlock()
}
