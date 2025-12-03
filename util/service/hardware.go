package service

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/server/service"
	serialports "go.bug.st/serial"
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /serial", getSerialPorts)

	service.Register("hardware", mux)
}

var (
	once  sync.Once
	ports []string
)

func getSerialPorts(w http.ResponseWriter, req *http.Request) {
	once.Do(func() {
		ports = strings.Split(os.Getenv("EVCC_SERIAL_PORTS"), ",")

		if len(ports) == 0 {
			ports, _ = serialports.GetPortsList()
		}
	})

	json.NewEncoder(w).Encode(ports)
}
