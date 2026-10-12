package smarthome

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/meter/fritz"
	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /devices", getDevices)

	service.Register("fritz", mux)
}

func getDevices(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	uri := strings.TrimRight(q.Get("uri"), "/")
	user := q.Get("user")
	password := q.Get("password")

	if uri == "" || user == "" || password == "" {
		jsonError(w, http.StatusBadRequest, errors.New("missing uri, user or password"))
		return
	}

	log := util.NewLogger("fritz").Redact(password)
	helper := request.NewHelper(log)
	helper.Client.Transport = request.NewTripper(log, transport.Insecure())

	settings := fritz.Settings{URI: uri, User: user, Password: password}

	sid, err := settings.GetSessionID(helper)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	if sid == "0000000000000000" {
		jsonError(w, http.StatusUnauthorized, errors.New("invalid user or password"))
		return
	}

	req, _ := request.New(http.MethodGet,
		fmt.Sprintf("%s/api/v0/smarthome/overview/devices", uri), nil, map[string]string{
			"Authorization": "AVM-SID " + sid,
		}, request.AcceptJSON,
	)

	var devices []Device
	if err := helper.DoJSON(req, &devices); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	w.Header().Set("Cache-control", "max-age=60")
	jsonWrite(w, deviceOptions(devices))
}

func deviceOptions(devices []Device) []service.Option {
	res := make([]service.Option, 0, len(devices))
	for _, d := range devices {
		// every device is a valid choice, a single one auto-fills the ain
		res = append(res, service.Option{
			Value: d.AIN,
			Label: strings.TrimSpace(d.Name),
			Hint:  strings.TrimSpace(d.ProductName),
			Match: true,
		})
	}
	// connected devices first, then by ain
	connected := func(o service.Option) bool {
		i := slices.IndexFunc(devices, func(d Device) bool { return d.AIN == o.Value })
		return devices[i].IsConnected
	}
	slices.SortFunc(res, func(a, b service.Option) int {
		if ca, cb := connected(a), connected(b); ca != cb {
			if ca {
				return -1
			}
			return 1
		}
		return cmp.Compare(a.Value, b.Value)
	})
	return res
}

func jsonWrite(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	jsonWrite(w, util.ErrorAsJson(err))
}
