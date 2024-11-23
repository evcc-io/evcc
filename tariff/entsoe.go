package tariff

import (
	"bytes"
	"encoding/xml"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"
	"fmt"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/entsoe"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

type Entsoe struct {
	*request.Helper
	*embed
	log    *util.Logger
	token  string
	domain string
	data   *util.Monitor[api.Rates]
	zones []struct {
			Price       float64
			Days, Hours string
		}
}

var _ api.Tariff = (*Entsoe)(nil)

func init() {
	registry.Add("entsoe", NewEntsoeFromConfig)
}

func NewEntsoeFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		embed         `mapstructure:",squash"`
		Securitytoken string
		Domain        string
		Zones []struct {
			Price       float64
			Days, Hours string
		}
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Securitytoken == "" {
		return nil, errors.New("missing securitytoken")
	}

	if cc.Domain == "" {
		return nil, errors.New("missing domain")
	}

	if err := cc.init(); err != nil {
		return nil, err
	}

	domain, err := entsoe.Area(entsoe.BZN, strings.ToUpper(cc.Domain))
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("entsoe").Redact(cc.Securitytoken)

	t := &Entsoe{
		log:    log,
		Helper: request.NewHelper(log),
		embed:  &cc.embed,
		token:  cc.Securitytoken,
		domain: domain,
		data:   util.NewMonitor[api.Rates](2 * time.Hour),
		zones: cc.Zones,
	}

	// Wrap the client with a decorator that adds the security token to each request.
	t.Client.Transport = &transport.Decorator{
		Base: t.Client.Transport,
		Decorator: transport.DecorateQuery(map[string]string{
			"securityToken": cc.Securitytoken,
		}),
	}

	done := make(chan error)
	go t.run(done)
	err = <-done

	return t, err
}

func (t *Entsoe) run(done chan error) {
	var once sync.Once

	// Data updated by ESO every half hour, but we only need data every hour to stay current.
	tick := time.NewTicker(time.Hour)
	for ; true; <-tick.C {
		var tr entsoe.PublicationMarketDocument

		if err := backoff.Retry(func() error {
			// Request the next 24 hours of data.
			data, err := t.DoBody(entsoe.DayAheadPricesRequest(t.domain, time.Hour*24))
			if err != nil {
				return backoffPermanentError(err)
			}

			var doc entsoe.Document
			if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&doc); err != nil {
				return backoff.Permanent(err)
			}

			switch doc.XMLName.Local {
			case entsoe.AcknowledgementMarketDocumentName:
				var doc entsoe.AcknowledgementMarketDocument
				if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&doc); err != nil {
					return backoff.Permanent(err)
				}

				return backoff.Permanent(errors.New(doc.Reason.Text))

			case entsoe.PublicationMarketDocumentName:
				if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&tr); err != nil {
					return backoff.Permanent(err)
				}

				if tr.Type != string(entsoe.ProcessTypeDayAhead) {
					return backoff.Permanent(errors.New("invalid document type: " + tr.Type))
				}

				return nil

			default:
				return backoff.Permanent(errors.New("invalid document name: " + doc.XMLName.Local))
			}
		}, bo()); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		if len(tr.TimeSeries) == 0 {
			once.Do(func() { done <- entsoe.ErrInvalidData })
			t.log.ERROR.Println(entsoe.ErrInvalidData)
			continue
		}

		// extract desired series
		res, err := entsoe.GetTsPriceData(tr.TimeSeries, entsoe.ResolutionHour)
		if err != nil {
			once.Do(func() { done <- err })
			t.log.ERROR.Println(err)
			continue
		}

		data := make(api.Rates, 0, len(res))
		for _, r := range res {
			var zonePrice float64 = 0.0

			weekDays := map[string]int{"Sun": 0, "Mon": 1, "Tue": 2, "Wed": 3, "Thu": 4, "Fri": 5, "Sat": 6}

			for _, zone := range t.zones{
				var dayStart, dayEnd string
				parsed, _ := fmt.Sscanf(strings.ReplaceAll(zone.Days,"-"," "), "%s %s", &dayStart, &dayEnd)

				// verify weekday parse process
				if parsed == 1 || parsed == 2 {
					// allow to specify single weekday
					if parsed == 1 {
						dayEnd = dayStart
					}

					if _, ok := weekDays[dayStart]; !ok {
						t.log.ERROR.Printf("Invalid weekday found: %s", dayStart)
						continue
					}

					if _, ok := weekDays[dayEnd]; !ok {
						t.log.ERROR.Printf("Invalid weekday found: %s", dayEnd)
						continue
					}
				} else {
					t.log.ERROR.Println("Invalid zone days: %s", zone.Days)
					continue
				}

				// skip zone if not part of weekday range
				if !(int(r.Start.Weekday()) >= weekDays[dayStart] && int(r.Start.Weekday()) <= weekDays[dayEnd]){
					continue
				}

				var hourStart, hourEnd int
				parsed, _ = fmt.Sscanf(zone.Hours, "%d-%d", &hourStart, &hourEnd)

				// verify hour parse process
				if parsed == 1 || parsed == 2 {
					// allow to specify single hour
					if parsed == 1 {
						hourEnd = hourStart
					}

					if (hourStart < 0 || hourStart > 23 || hourEnd < 0 || hourEnd > 23){
						t.log.ERROR.Printf("Invalid hour range found, hourStart or hourEnd must be within 0-23 range: %d-%d", hourStart, hourEnd)
						continue
					}

					if (hourStart > hourEnd){
						t.log.ERROR.Printf("Invalid hour range found, hourStart cannot be higher than hourEnd: %d-%d", hourStart, hourEnd)
						continue
					}

				} else {
					t.log.ERROR.Println("Invalid zone hours: %s", zone.Hours)
					continue
				}


				// skip zone if not part of hour range
				if !(int(r.Start.Hour()) >= hourStart && int(r.Start.Hour()) <= hourEnd){
					continue
				}

				t.log.DEBUG.Printf("Found maching zone %s for %s with currentPrice: %f", zone, r.Start, t.totalPrice(r.Value))

				zonePrice = zone.Price
				break
			}

			ar := api.Rate{
				Start: r.Start,
				End:   r.End,
				Price: t.totalPrice(r.Value) + zonePrice,
			}

			data = append(data, ar)
		}

		mergeRates(t.data, data)
		once.Do(func() { close(done) })
	}
}

// Rates implements the api.Tariff interface
func (t *Entsoe) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *Entsoe) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
