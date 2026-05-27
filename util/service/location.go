package service

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"

	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/spf13/cast"
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /lat", update(getLatitude))
	mux.HandleFunc("GET /lon", update(getLongitude))
	mux.HandleFunc("GET /ip", update(getIP))

	service.Register("location", mux)
}

type IpApi struct {
	CountryCode string
	City        string
	Zip         string
	Lat         float64
	Lon         float64
	Query       net.IP
}

var (
	onceLocation sync.Once
	location     IpApi
)

func update(fun func(http.ResponseWriter, *http.Request)) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		onceLocation.Do(func() {
			log := util.NewLogger("main")
			client := request.NewHelper(log)
			if err := client.GetJSON("http://ip-api.com/json", &location); err != nil {
				log.ERROR.Printf("location: %v", err)
			}
		})

		fun(w, req)
	}
}

func getLatitude(w http.ResponseWriter, req *http.Request) {
	json.NewEncoder(w).Encode([]string{cast.ToString(location.Lat)})
}

func getLongitude(w http.ResponseWriter, req *http.Request) {
	json.NewEncoder(w).Encode([]string{cast.ToString(location.Lon)})
}

func getIP(w http.ResponseWriter, req *http.Request) {
	json.NewEncoder(w).Encode([]string{location.Query.String()})
}
