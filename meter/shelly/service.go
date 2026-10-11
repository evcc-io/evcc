package shelly

import (
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
	"github.com/samber/lo"
)

var (
	log            = util.NewLogger("shelly")
	redactPassword = log.RotatingSlot()
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /channels", getChannels)

	service.Register("shelly", mux)
}

// channel is a relay/meter slot of a device, merged across components sharing the same id
type channel struct {
	id      int
	name    string
	relay   bool
	metered bool    // has a power meter
	power   float64 // W, valid if metered
}

// gen2Components are the Shelly.GetStatus components that make up a channel
var gen2Components = []string{"switch", "pm1", "em1", "em"}

// channelOptions maps channels to options. With metered only channels with a power
// meter match (meter templates), otherwise only relays do (charger templates).
func channelOptions(channels []channel, metered bool) []service.Option {
	res := make([]service.Option, 0, len(channels))
	for _, ch := range channels {
		kind := "Meter"
		if ch.relay {
			kind = "Switch"
		}

		o := service.Option{
			Value: strconv.Itoa(ch.id),
			Label: cmp.Or(strings.TrimSpace(ch.name), fmt.Sprintf("%s %d", kind, ch.id+1)),
			Match: ch.relay,
		}
		if metered {
			o.Match = ch.metered
		}
		if ch.metered {
			o.Hint = fmt.Sprintf("%.0f W", ch.power)
		}
		res = append(res, o)
	}

	return res
}

// gen1Channels lists the relays and (e)meters of a /status response by index
func gen1Channels(res Gen1Status) []channel {
	var channels []channel
	for i := range max(len(res.Relays), len(res.Meters), len(res.EMeters)) {
		ch := channel{id: i, relay: i < len(res.Relays)}
		switch {
		case i < len(res.Meters):
			ch.metered, ch.power = true, res.Meters[i].Power
		case i < len(res.EMeters):
			ch.metered, ch.power = true, res.EMeters[i].Power
		}
		channels = append(channels, ch)
	}
	return channels
}

// gen2Channels merges the switch/pm1/em1/em components of Shelly.GetStatus and
// Shelly.GetConfig by id. Switches without power metering omit apower.
func gen2Channels(status, config map[string]json.RawMessage) []channel {
	byId := make(map[int]channel)

	// sorted keys keep the merged name deterministic (em1:0 before switch:0)
	for _, key := range slices.Sorted(maps.Keys(status)) {
		typ, idStr, ok := strings.Cut(key, ":")
		if !ok {
			continue
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			continue
		}

		if !slices.Contains(gen2Components, typ) {
			continue
		}

		var res struct {
			Apower        *float64 `json:"apower"`
			ActPower      *float64 `json:"act_power"`
			TotalActPower *float64 `json:"total_act_power"`
		}
		var cfg struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(status[key], &res)
		_ = json.Unmarshal(config[key], &cfg)

		ch := byId[id]
		ch.id = id
		ch.relay = ch.relay || typ == "switch"
		if power := cmp.Or(res.Apower, res.ActPower, res.TotalActPower); power != nil {
			ch.metered, ch.power = true, *power
		}
		if strings.TrimSpace(ch.name) == "" {
			ch.name = cfg.Name
		}
		byId[id] = ch
	}

	res := lo.Values(byId)
	slices.SortFunc(res, func(a, b channel) int { return a.id - b.id })
	return res
}

func getChannels(w http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	password := q.Get("password")
	metered, _ := strconv.ParseBool(q.Get("metered"))

	// the logger is shared, a rotating slot keeps the redaction list from growing while typing
	redactPassword(password)

	res, err := deviceChannels(q.Get("uri"), q.Get("user"), password)
	if err != nil {
		// form incomplete or device unreachable, the ui polls while typing
		log.DEBUG.Printf("channels: %v", err)
		res = nil
	}

	// short cache: hints carry live values, the form refetches on every change
	w.Header().Set("Cache-control", "max-age=30")
	jsonWrite(w, channelOptions(res, metered))
}

func deviceChannels(uri, user, password string) ([]channel, error) {
	client, uri, info, err := newClient(log, uri, user, password)
	if err != nil {
		return nil, err
	}

	if info.Gen < 2 {
		var res Gen1Status
		if err := client.GetJSON(uri+"/status", &res); err != nil {
			return nil, err
		}
		return gen1Channels(res), nil
	}

	var status, config map[string]json.RawMessage
	if err := client.GetJSON(uri+"/rpc/Shelly.GetStatus", &status); err != nil {
		return nil, err
	}
	if err := client.GetJSON(uri+"/rpc/Shelly.GetConfig", &config); err != nil {
		return nil, err
	}

	return gen2Channels(status, config), nil
}

func jsonWrite(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
