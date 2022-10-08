//go:build !windows

package server

import (
	"errors"
	"net"
	"net/http"
	"os"

	"github.com/evcc-io/evcc/core/site"
)

// SocketPath is the unix domain socket path
const SocketPath = "/tmp/evcc"

// removeIfExists deletes file if it exists or fails
func removeIfExists(file string) {
	_, err := os.Stat(file)
	if err == nil {
		err = os.Remove(file)
	}

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.FATAL.Fatal(err)
	}
}

// HealthListener attaches listener to unix domain socket and runs listener
func HealthListener(site site.API, exitC <-chan struct{}) {
	removeIfExists(SocketPath)

	l, err := net.Listen("unix", SocketPath)
	if err != nil {
		log.FATAL.Fatal(err)
	}
	defer l.Close()

	mux := http.NewServeMux()
	httpd := http.Server{Handler: mux}
	mux.HandleFunc("/health", healthHandler(site))

	go func() { _ = httpd.Serve(l) }()

	<-exitC
	removeIfExists(SocketPath) // cleanup
}
