//go:build !windows

package server

import (
	"net"
	"net/http"
	"os"

	"github.com/evcc-io/evcc/cmd/shutdown"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/util"
)

// SocketPath is the unix domain socket path
const SocketPath = "/tmp/evcc"

// removeIfExists deletes file if it exists or fails
func removeIfExists(file string) error {
	if _, err := os.Stat(file); err == nil {
		if err := os.RemoveAll(file); err != nil {
			return err
		}
	}
	return nil
}

// HealthListener attaches listener to unix domain socket and runs listener
func HealthListener(site site.API) {
	log := util.NewLogger("main")

	err := removeIfExists(SocketPath)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	l, err := net.Listen("unix", SocketPath)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	mux := http.NewServeMux()
	httpd := http.Server{Handler: mux}
	mux.HandleFunc("/health", healthHandler(site))

	go func() { _ = httpd.Serve(l) }()

	shutdown.Register(func() {
		_ = l.Close()
		_ = removeIfExists(SocketPath) // cleanup
	})
}
