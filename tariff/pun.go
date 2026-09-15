package tariff

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// ErrPunDataNotAvailable indicates that GME returned no data for the requested date range (HTTP 404).
var ErrPunDataNotAvailable = errors.New("PUN data not available")

// romeLocation is resolved once at package init to avoid repeated filesystem lookups.
var romeLocation *time.Location

type Pun struct {
	*embed
	zone     string
	history  int
	forecast int
	log      *util.Logger
	data     *util.Monitor[api.Rates]
}

type NewDataSet struct {
	XMLName xml.Name `xml:"NewDataSet"`
	Prezzi  []Prezzo `xml:"Prezzi"`
}

type Zone struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

type Prezzo struct {
	Data  string `xml:"Data"`
	Ora   string `xml:"Ora"`
	Zones []Zone `xml:",any"`
}

var _ api.Tariff = (*Pun)(nil)

func init() {
	registry.Add("pun", NewPunFromConfig)
	romeLocation, _ = time.LoadLocation("Europe/Rome")
}

func NewPunFromConfig(other map[string]any) (api.Tariff, error) {
	var cc struct {
		embed    `mapstructure:",squash"`
		Zone     string
		History  int
		Forecast int
	}

	logger := util.NewLogger("pun")

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if err := cc.init(); err != nil {
		return nil, err
	}

	var zone = strings.ToUpper(strings.TrimSpace(cc.Zone))
	if cc.Zone == "" {
		zone = "PUN"
	}

	if cc.History <= 0 {
		cc.History = 30
	}

	t := &Pun{
		log:      logger,
		zone:     zone,
		history:  cc.History,
		forecast: cc.Forecast,
		embed:    &cc.embed,
		data:     util.NewMonitor[api.Rates](2 * time.Hour),
	}

	return runOrError(t)
}

func (t *Pun) run(done chan error) {
	var once sync.Once

	for tick := time.Tick(time.Hour); ; <-tick {
		now := time.Now().In(romeLocation)
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, romeLocation)

		start := today
		if t.forecast > 0 {
			start = today.AddDate(0, 0, -t.history)
		}

		res, err := backoff.RetryWithData(func() (api.Rates, error) {
			res, err := t.getData(start, today.AddDate(0, 0, 1))
			return res, backoffPermanentError(err)
		}, bo())
		if err != nil {
			if reportError(&once, done, err) {
				return
			}
			t.log.ERROR.Println(err)
			continue
		}

		if t.forecast > 0 {
			res = t.extendForecast(res, today)
		}

		mergeRates(t.data, res)
		once.Do(func() { close(done) })
	}
}

func (t *Pun) extendForecast(rates api.Rates, today time.Time) api.Rates {
	if len(rates) == 0 {
		return rates
	}

	profile := newPriceProfile(rates)
	lastEnd := rates[len(rates)-1].End

	rates = slices.DeleteFunc(rates, func(r api.Rate) bool {
		return r.Start.Before(today)
	})

	horizon := lastEnd.AddDate(0, 0, t.forecast)
	for ts := lastEnd; ts.Before(horizon); ts = ts.Add(time.Hour) {
		if price, ok := profile.price(ts); ok {
			rates = append(rates, api.Rate{Start: ts, End: ts.Add(time.Hour), Value: price})
		}
	}

	rates.Sort()
	return rates
}

// Rates implements the api.Tariff interface
func (t *Pun) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *Pun) Type() api.TariffType {
	return api.TariffTypePriceForecast
}

func (t *Pun) priceForZone(p Prezzo) (float64, error) {
	for _, z := range p.Zones {
		if strings.ToUpper(strings.TrimSpace(z.XMLName.Local)) == t.zone {
			price, err := strconv.ParseFloat(strings.ReplaceAll(z.Value, ",", "."), 64)
			return price, err
		}
	}
	return 0, fmt.Errorf("zone %s not found for hour %s", t.zone, p.Ora)
}

func (t *Pun) getData(start, end time.Time) (api.Rates, error) {
	client := request.NewClient(t.log)
	client.Jar, _ = cookiejar.New(nil)

	// Request the ZIP file
	uri := "https://gme.mercatoelettrico.org/DesktopModules/GmeDownload/API/ExcelDownload/downloadzipfile?DataInizio=" + start.Format("20060102") + "&DataFine=" + end.Format("20060102") + "&Date=" + start.Format("20060102") + "&Mercato=MGP&Settore=Prezzi&FiltroDate=InizioFine"
	req, _ := http.NewRequest("GET", uri, nil)
	req.Header = http.Header{
		"Referer":            {"https://gme.mercatoelettrico.org/en-us/Home/Results/Electricity/MGP/Download?valore=Prezzi"},
		"moduleid":           {"12103"},
		"sec-ch-ua-mobile":   {"?0"},
		"sec-ch-ua-platform": {"Windows"},
		"sec-fetch-dest":     {"empty"},
		"sec-fetch-mode":     {"cors"},
		"sec-fetch-site":     {"same-origin"},
		"sec-gpc":            {"1"},
		"tabid":              {"1749"},
		"userid":             {"-1"},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: %s..%s", ErrPunDataNotAvailable, start.Format("2006-01-02"), end.Format("2006-01-02"))
	}

	body, err := request.ReadBody(resp)
	if err != nil {
		return nil, err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, err
	}

	var data api.Rates
	for _, file := range zipReader.File {
		if !strings.HasSuffix(file.Name, "Prezzi.xml") {
			continue
		}

		f, err := file.Open()
		if err != nil {
			return nil, err
		}

		var dataSet NewDataSet
		err = xml.NewDecoder(f).Decode(&dataSet)
		f.Close()
		if err != nil {
			return nil, err
		}

		rates, err := t.parseDataSet(dataSet)
		if err != nil {
			return nil, err
		}
		data = append(data, rates...)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no tariff data in downloaded ZIP archive")
	}

	data.Sort()
	return data, nil
}

func (t *Pun) parseDataSet(dataSet NewDataSet) (api.Rates, error) {
	data := make(api.Rates, 0, len(dataSet.Prezzi))

	for _, p := range dataSet.Prezzi {
		date, err := time.Parse("20060102", p.Data)
		if err != nil {
			return nil, fmt.Errorf("parse date: %w", err)
		}

		hour, err := strconv.Atoi(p.Ora)
		if err != nil {
			return nil, fmt.Errorf("parse hour: %w", err)
		}

		// Adjust hour to handle edge case where p.Ora is "00"
		if hour == 0 {
			hour = 24
			date = date.AddDate(0, 0, -1)
		}

		price, err := t.priceForZone(p)
		if err != nil {
			return nil, fmt.Errorf("parse price: %w", err)
		}

		ts := time.Date(date.Year(), date.Month(), date.Day(), hour-1, 0, 0, 0, romeLocation)
		data = append(data, api.Rate{
			Start: ts,
			End:   ts.Add(time.Hour),
			Value: t.totalPrice(price/1e3, ts),
		})
	}

	return data, nil
}

type priceProfile struct {
	sum   [2][24]float64
	count [2][24]int
}

func isWeekend(ts time.Time) bool {
	return ts.Weekday() == time.Saturday || ts.Weekday() == time.Sunday
}

func newPriceProfile(rates api.Rates) (p priceProfile) {
	for _, r := range rates {
		dt := 0
		if isWeekend(r.Start) {
			dt = 1
		}
		p.sum[dt][r.Start.Hour()] += r.Value
		p.count[dt][r.Start.Hour()]++
	}
	return p
}

func (p priceProfile) price(ts time.Time) (float64, bool) {
	dt := 0
	if isWeekend(ts) {
		dt = 1
	}

	hour := ts.Hour()
	// fall back to the other day class to avoid gaps for sparse history
	if p.count[dt][hour] == 0 {
		dt = 1 - dt
		if p.count[dt][hour] == 0 {
			return 0, false
		}
	}
	return p.sum[dt][hour] / float64(p.count[dt][hour]), true
}
