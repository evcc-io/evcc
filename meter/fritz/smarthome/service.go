package smarthome

import (
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
	"github.com/samber/lo"
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

	ains := lo.Map(devices, func(d Device, _ int) string {
		return d.AIN
	})
	slices.Sort(ains)

	w.Header().Set("Cache-control", "max-age=60")
	jsonWrite(w, ains)
}

func jsonWrite(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	jsonWrite(w, util.ErrorAsJson(err))
}
