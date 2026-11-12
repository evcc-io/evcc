package tasmota

import (
	"cmp"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /channels", getChannels)

	service.Register("tasmota", mux)
}

// statusResponse combines the Status and StatusSTS parts of the Status 0 response
type statusResponse struct {
	StatusResponse
	StatusSTSResponse
}

// relays returns the state of relays 1..8, empty when the relay does not exist
func (res StatusSTSResponse) relays() [8]string {
	sts := res.StatusSTS
	return [8]string{cmp.Or(sts.Power1, sts.Power), sts.Power2, sts.Power3, sts.Power4, sts.Power5, sts.Power6, sts.Power7, sts.Power8}
}

// channelOptions lists relay and energy channels. Relays match for chargers, channels with energy readings for meters.
func channelOptions(st statusResponse, sns StatusSNSResponse, meter bool) []service.Option {
	relays := st.relays()
	power := sns.StatusSNS.Energy.Power

	var res []service.Option
	for ch := 1; ch <= 8; ch++ {
		relay := relays[ch-1] != ""
		energy := ch <= len(power)
		if !relay && !energy {
			continue
		}

		match := relay
		if meter {
			match = energy
		}

		o := service.Option{
			Value: strconv.Itoa(ch),
			Label: fmt.Sprintf("Channel %d", ch),
			Match: match,
		}
		if names := st.Status.FriendlyName; ch <= len(names) {
			o.Label = cmp.Or(strings.TrimSpace(names[ch-1]), o.Label)
		}
		if energy {
			o.Hint = fmt.Sprintf("%.0f W", power[ch-1])
		}

		res = append(res, o)
	}

	return res
}

// channels queries the device; unreachable or unauthorized devices yield no options
func channels(uri, user, password string, meter bool) []service.Option {
	// NewConnection redacts the credentials on the shared logger
	log := util.NewLogger("tasmota")

	// channel is irrelevant for discovery
	conn, err := NewConnection(uri, user, password, []int{1}, 0)
	if err != nil {
		log.DEBUG.Println(err)
		return nil
	}

	var st statusResponse
	if err := conn.cmd("Status 0", &st); err != nil {
		log.DEBUG.Println(err)
		return nil
	}

	var sns StatusSNSResponse
	if err := conn.cmd("Status 8", &sns); err != nil {
		log.DEBUG.Println(err)
		return nil
	}

	return channelOptions(st, sns, meter)
}

func getChannels(w http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	meter, _ := strconv.ParseBool(q.Get("meter"))

	var res []service.Option
	if uri := q.Get("uri"); uri != "" {
		res = channels(uri, q.Get("user"), q.Get("password"), meter)
	}
	if res == nil {
		res = []service.Option{}
	}

	// short cache: hints carry live values, the form refetches on every change
	w.Header().Set("Cache-control", "max-age=30")
	jsonWrite(w, res)
}

func jsonWrite(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
