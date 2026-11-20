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
	ports []service.Option
)

func getSerialPorts(w http.ResponseWriter, req *http.Request) {
	once.Do(func() {
		if env := os.Getenv("EVCC_SERIAL_PORTS"); env != "" {
			ports = plainOptions(strings.Split(env, ","))
		} else {
			ports = serialPortOptions()
		}
	})

	json.NewEncoder(w).Encode(ports)
}

// plainOptions wraps port names as options without details
func plainOptions(names []string) []service.Option {
	res := make([]service.Option, 0, len(names))
	for _, name := range names {
		res = append(res, service.Option{Value: name, Match: true})
	}
	return res
}

// plainSerialPorts lists port names without USB details
func plainSerialPorts() []service.Option {
	names, _ := serialports.GetPortsList()
	return plainOptions(names)
}
