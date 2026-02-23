package tariff

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/homeassistant"
)

func init() {
	registry.Add("homeassistant", NewHATariffFromConfig)
}

// haStateSource abstracts HA state fetching for testability
type haStateSource interface {
	GetFloatState(entity string) (float64, error)
	GetJSON(uri string, result any) error
}

// HATariff reads electricity prices or solar forecast from Home Assistant entities
type HATariff struct {
	*embed
	log        *util.Logger
	source     haStateSource
	baseURI    string
	entities   []string
	attributes []string
	format     string
	priceG     func() (float64, error)
	data       *util.Monitor[api.Rates]
	typ        api.TariffType
}

var _ api.Tariff = (*HATariff)(nil)

func NewHATariffFromConfig(other map[string]any) (api.Tariff, error) {
	cc := struct {
		embed      `mapstructure:",squash"`
		URI        string
		Entities   []string // one or more entity IDs to merge (e.g. today, d1, d2, d3)
		Attributes []string // attributes to read from each entity
		Format     string   // "nordpool" (native), "zonneplan", "entsoe", "watts", "solcast"
		Tariff     api.TariffType `mapstructure:"tariff"`
		Interval   time.Duration
		Cache      time.Duration
	}{
		Interval: time.Hour,
		Cache:    5 * time.Minute,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	if len(cc.Entities) == 0 {
		return nil, errors.New("missing entities")
	}

	if err := cc.init(); err != nil {
		return nil, err
	}

	log := util.NewLogger("homeassistant")

	conn, err := homeassistant.NewConnection(log, cc.URI, "")
	if err != nil {
		return nil, err
	}

	baseURI := util.DefaultScheme(strings.TrimSuffix(cc.URI, "/"), "http")

	return newHATariff(&cc.embed, log, conn, baseURI, cc.Entities, cc.Attributes, cc.Format, cc.Tariff, cc.Interval, cc.Cache)
}

func newHATariff(
	emb *embed,
	log *util.Logger,
	source haStateSource,
	baseURI string,
	entities []string,
	attributes []string,
	format string,
	typ api.TariffType,
	interval, cache time.Duration,
) (api.Tariff, error) {
	t := &HATariff{
		embed:      emb,
		log:        log,
		source:     source,
		baseURI:    baseURI,
		entities:   entities,
		attributes: attributes,
		format:     format,
		typ:        typ,
		data:       util.NewMonitor[api.Rates](2 * interval),
	}

	if len(attributes) == 0 {
		// price mode: only a single entity makes sense
		if len(entities) != 1 {
			return nil, errors.New("price mode requires exactly one entity")
		}
		priceG := func() (float64, error) {
			v, err := source.GetFloatState(entities[0])
			if err != nil {
				t.log.WARN.Printf("entity %s: %v", entities[0], err)
			}
			return v, err
		}
		t.priceG = util.Cached(priceG, cache)
		return t, nil
	}

	// forecast mode: background loop over all entities and attributes
	return runOrError(t)
}

func (t *HATariff) run(done chan error) {
	var once sync.Once

	for tick := time.Tick(time.Hour); ; <-tick {
		if err := backoff.Retry(func() error {
			var combined api.Rates

			for _, entity := range t.entities {
				uri := fmt.Sprintf("%s/api/states/%s", t.baseURI, url.PathEscape(entity))

				var res struct {
					Attributes map[string]json.RawMessage `json:"attributes"`
				}

				if err := t.source.GetJSON(uri, &res); err != nil {
					return backoffPermanentError(fmt.Errorf("entity %s: %w", entity, err))
				}

				for _, attr := range t.attributes {
					raw, ok := res.Attributes[attr]
					if !ok {
						t.log.DEBUG.Printf("attribute %q not found on %s, skipping", attr, entity)
						continue
					}

					rates, err := t.parseRates(raw)
					if err != nil {
						return backoff.Permanent(fmt.Errorf("%s attribute %q: %w", entity, attr, err))
					}
					combined = append(combined, rates...)
				}
			}

			for i, r := range combined {
				combined[i] = api.Rate{
					Value: t.totalPrice(r.Value, r.Start),
					Start: r.Start.Local(),
					End:   r.End.Local(),
				}
			}

			if len(combined) == 0 {
				t.log.WARN.Println("no rates collected from any entity/attribute")
			}
			mergeRates(t.data, combined)
			return nil
		}, bo()); err != nil {
			once.Do(func() { done <- err })
			t.log.ERROR.Println(err)
			continue
		}

		once.Do(func() { close(done) })
	}
}

// parseRates converts a raw JSON attribute to api.Rates using the configured format
func (t *HATariff) parseRates(raw json.RawMessage) (api.Rates, error) {
	switch t.format {
	case "zonneplan":
		return parseZonneplan(raw)
	case "entsoe":
		return parseEntsoe(raw)
	case "watts":
		return parseWatts(raw)
	case "solcast":
		return parseSolcast(raw)
	default: // "" or "nordpool" — native api.Rates JSON (start/end/value)
		var rates api.Rates
		if err := json.Unmarshal(raw, &rates); err != nil {
			return nil, err
		}
		return rates, nil
	}
}

// zonneplanEntry is one slot from the Zonneplan forecast attribute
type zonneplanEntry struct {
	Datetime         string `json:"datetime"`
	ElectricityPrice int64  `json:"electricity_price"`
}

// parseZonneplan converts Zonneplan forecast entries to api.Rates.
// electricity_price is in units of 1e-7 €/kWh.
// Slot end is inferred from the next entry's start; the last slot is 1h.
func parseZonneplan(raw json.RawMessage) (api.Rates, error) {
	var entries []zonneplanEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}

	rates := make(api.Rates, len(entries))
	for i, e := range entries {
		start, err := time.Parse(time.RFC3339Nano, e.Datetime)
		if err != nil {
			return nil, fmt.Errorf("datetime %q: %w", e.Datetime, err)
		}

		var end time.Time
		if i+1 < len(entries) {
			end, err = time.Parse(time.RFC3339Nano, entries[i+1].Datetime)
			if err != nil {
				return nil, fmt.Errorf("datetime %q: %w", entries[i+1].Datetime, err)
			}
		} else {
			end = start.Add(time.Hour)
		}

		rates[i] = api.Rate{
			Start: start,
			End:   end,
			Value: float64(e.ElectricityPrice) / 1e7,
		}
	}

	return rates, nil
}

// entsoeEntry is one slot from the ENTSO-E HA integration attribute (prices_today / prices_tomorrow)
type entsoeEntry struct {
	Time  string  `json:"time"`
	Price float64 `json:"price"`
}

// entsoeTimeFormat matches "2026-02-23 00:00:00+01:00"
const entsoeTimeFormat = "2006-01-02 15:04:05-07:00"

// parseEntsoe converts ENTSO-E HA integration entries to api.Rates.
// Slot end is inferred from the next entry's time; the last slot is 1h.
func parseEntsoe(raw json.RawMessage) (api.Rates, error) {
	var entries []entsoeEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}

	rates := make(api.Rates, len(entries))
	for i, e := range entries {
		start, err := time.Parse(entsoeTimeFormat, e.Time)
		if err != nil {
			return nil, fmt.Errorf("time %q: %w", e.Time, err)
		}

		var end time.Time
		if i+1 < len(entries) {
			end, err = time.Parse(entsoeTimeFormat, entries[i+1].Time)
			if err != nil {
				return nil, fmt.Errorf("time %q: %w", entries[i+1].Time, err)
			}
		} else {
			end = start.Add(time.Hour)
		}

		rates[i] = api.Rate{
			Start: start,
			End:   end,
			Value: e.Price,
		}
	}

	return rates, nil
}

// parseWatts converts a {ISO-8601 timestamp → watts} dict attribute to api.Rates.
// Used by Forecast.Solar and Open-Meteo HA integrations (attribute: watts), 15-min slots.
func parseWatts(raw json.RawMessage) (api.Rates, error) {
	var data map[string]float64
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}

	keys := slices.Sorted(maps.Keys(data))
	rates := make(api.Rates, len(keys))

	for i, k := range keys {
		start, err := time.Parse(time.RFC3339, k)
		if err != nil {
			return nil, fmt.Errorf("timestamp %q: %w", k, err)
		}

		var end time.Time
		if i+1 < len(keys) {
			if end, err = time.Parse(time.RFC3339, keys[i+1]); err != nil {
				return nil, fmt.Errorf("timestamp %q: %w", keys[i+1], err)
			}
		} else {
			end = start.Add(SlotDuration)
		}

		rates[i] = api.Rate{
			Start: start,
			End:   end,
			Value: data[k],
		}
	}

	return rates, nil
}

// solcastEntry is one 30-min slot from the Solcast HA integration detailedForecast attribute
type solcastEntry struct {
	PeriodStart string  `json:"period_start"`
	PvEstimate  float64 `json:"pv_estimate"`
}

// parseSolcast converts Solcast detailedForecast entries to api.Rates.
// pv_estimate is in kW; slots are fixed at 30 minutes.
func parseSolcast(raw json.RawMessage) (api.Rates, error) {
	var entries []solcastEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	rates := make(api.Rates, len(entries))
	for i, e := range entries {
		start, err := time.Parse(time.RFC3339, e.PeriodStart)
		if err != nil {
			return nil, fmt.Errorf("period_start %q: %w", e.PeriodStart, err)
		}
		rates[i] = api.Rate{
			Start: start,
			End:   start.Add(30 * time.Minute),
			Value: e.PvEstimate * 1000, // kW → W
		}
	}
	return rates, nil
}

func (t *HATariff) priceRates() (api.Rates, error) {
	price, err := t.priceG()
	if err != nil {
		return nil, err
	}

	res := make(api.Rates, 48*4) // 2 days of 15-min slots
	start := time.Now().Truncate(SlotDuration)

	for i := range res {
		slot := start.Add(time.Duration(i) * SlotDuration)
		res[i] = api.Rate{
			Start: slot,
			End:   slot.Add(SlotDuration),
			Value: t.totalPrice(price, slot),
		}
	}

	return res, nil
}

// Rates implements the api.Tariff interface
func (t *HATariff) Rates() (api.Rates, error) {
	if t.priceG != nil {
		return t.priceRates()
	}

	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *HATariff) Type() api.TariffType {
	switch {
	case t.typ != 0:
		return t.typ
	case t.priceG != nil:
		return api.TariffTypePriceDynamic
	default:
		return api.TariffTypePriceForecast
	}
}
