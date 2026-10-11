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

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /channels", getChannels)

	service.Register("shelly", mux)
}

// channel is a relay/meter slot of a device, merged across components sharing the same id
type channel struct {
	id    int
	name  string
	relay bool
	power *float64
}

// channelOptions maps channels to options. With metered only channels with a power
// meter match (meter templates), otherwise only relays do (charger templates).
func channelOptions(channels []channel, metered bool) []service.Option {
	res := make([]service.Option, 0, len(channels))
	for _, ch := range channels {
		o := service.Option{
			Value: strconv.Itoa(ch.id),
			Label: strings.TrimSpace(ch.name),
			Match: ch.relay,
		}
		if metered {
			o.Match = ch.power != nil
		}
		if o.Label == "" {
			o.Label = fmt.Sprintf("Meter %d", ch.id+1)
			if ch.relay {
				o.Label = fmt.Sprintf("Switch %d", ch.id+1)
			}
		}
		if ch.power != nil {
			o.Hint = fmt.Sprintf("%.0f W", *ch.power)
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
			ch.power = lo.ToPtr(res.Meters[i].Power)
		case i < len(res.EMeters):
			ch.power = lo.ToPtr(res.EMeters[i].Power)
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

		if !slices.Contains([]string{"switch", "pm1", "em1", "em"}, typ) {
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
			ch.power = power
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

	log := util.NewLogger("shelly").Redact(password)

	res, err := deviceChannels(log, q.Get("uri"), q.Get("user"), password)
	if err != nil {
		// form incomplete or device unreachable, the ui polls while typing
		log.DEBUG.Printf("channels: %v", err)
		res = nil
	}

	// short cache: hints carry live values, the form refetches on every change
	w.Header().Set("Cache-control", "max-age=30")
	jsonWrite(w, channelOptions(res, metered))
}

func deviceChannels(log *util.Logger, uri, user, password string) ([]channel, error) {
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
