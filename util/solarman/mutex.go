package solarman

import "sync"

var mutex sync.Mutex

func Lock() {
	mutex.Lock()
}

func Unlock() {
	mutex.Unlock()
}
