//go:build !windows

package server

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/evcc-io/evcc/cmd/shutdown"
	"github.com/evcc-io/evcc/core/site"
)

// socketPath is the unix domain socket path
const socketPath = "/tmp/evcc-%d"

// SocketPath returns the unix domain socket path for a given port
func SocketPath(port int) string {
	return fmt.Sprintf(socketPath, port)
}

// removeIfExists deletes file if it exists or fails
func removeIfExists(file string) {
	if _, err := os.Stat(file); err == nil {
		if err := os.RemoveAll(file); err != nil {
			log.FATAL.Fatal(err)
		}
	}
}

// HealthListener attaches listener to unix domain socket and runs listener
func HealthListener(site site.API, port int) {
	socket := SocketPath(port)

	removeIfExists(socket)

	l, err := net.Listen("unix", socket)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	mux := http.NewServeMux()
	httpd := http.Server{Handler: mux}
	mux.HandleFunc("/health", healthHandler(site))

	go func() { _ = httpd.Serve(l) }()

	shutdown.Register(func() {
		_ = l.Close()
		removeIfExists(socket) // cleanup
	})
}
