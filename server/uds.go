//go:build !windows

package server

import (
	"errors"
	"net"
	"net/http"
	"os"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/util/log"
)

// socketPath is the unix domain socket path
const socketPath = "/tmp/evcc"

// removeIfExists deletes file if it exists or fails
func removeIfExists(file string) {
	_, err := os.Stat(file)
	if err == nil {
		err = os.Remove(file)
	}

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error("cannot remove %s: %v", socketPath, err)
		os.Exit(1)
	}
}

// HealthListener attaches listener to unix domain socket and runs listener
func HealthListener(site site.API, exitC <-chan struct{}) {
	removeIfExists(socketPath)

	l, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Error("listen to unix://%s: %v", socketPath, err)
		os.Exit(1)
	}
	defer l.Close()

	mux := http.NewServeMux()
	httpd := http.Server{Handler: mux}
	mux.HandleFunc("/health", healthHandler(site))

	go func() { _ = httpd.Serve(l) }()

	<-exitC
	removeIfExists(socketPath) // cleanup
}
