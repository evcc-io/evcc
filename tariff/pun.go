package tariff

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
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

type Pun struct {
	*embed
	log  *util.Logger
	data *util.Monitor[api.Rates]
}

type NewDataSet struct {
	XMLName xml.Name `xml:"NewDataSet"`
	Prezzi  []Prezzo `xml:"Prezzi"`
}

type Prezzo struct {
	Data string `xml:"Data"`
	Ora  string `xml:"Ora"`
	PUN  string `xml:"PUN"`
}

type Rate struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Price float64   `json:"price"`
}

var _ api.Tariff = (*Pun)(nil)

func init() {
	registry.Add("pun", NewPunFromConfig)
}

func NewPunFromConfig(other map[string]any) (api.Tariff, error) {
	var cc embed

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if err := cc.init(); err != nil {
		return nil, err
	}

	t := &Pun{
		log:   util.NewLogger("pun"),
		embed: &cc,
		data:  util.NewMonitor[api.Rates](2 * time.Hour),
	}

	return runOrError(t)
}

func (t *Pun) run(done chan error) {
	var once sync.Once

	for tick := time.Tick(time.Hour); ; <-tick {
		// get today data
		today, err := backoff.RetryWithData(func() (api.Rates, error) {
			res, err := t.getData(time.Now())
			return res, backoffPermanentError(err)
		}, bo())
		if err != nil {
			once.Do(func() { done <- err })
			t.log.ERROR.Println(err)
			continue
		}

		// get tomorrow data
		res, err := backoff.RetryWithData(func() (api.Rates, error) {
			res, err := t.getData(time.Now().AddDate(0, 0, 1))
			return res, backoffPermanentError(err)
		}, bo())
		if err != nil {
			once.Do(func() { done <- err })
			t.log.ERROR.Println(err)
			continue
		}

		// merge today and tomorrow data
		data := append(today, res...)

		mergeRates(t.data, data)
		once.Do(func() { close(done) })
	}
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

func (t *Pun) getData(day time.Time) (api.Rates, error) {
	client := request.NewClient(t.log)
	client.Jar, _ = cookiejar.New(nil)

	// Request the ZIP file
	uri := "https://gme.mercatoelettrico.org/DesktopModules/GmeDownload/API/ExcelDownload/downloadzipfile?DataInizio=" + day.Format("20060102") + "&DataFine=" + day.Format("20060102") + "&Date=" + day.Format("20060102") + "&Mercato=MGP&Settore=Prezzi&FiltroDate=InizioFine"
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
	if err != nil || resp.StatusCode == http.StatusNotFound {
		return nil, err
	}

	body, err := request.ReadBody(resp)
	if err != nil {
		return nil, err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, err
	}

	var tariffFile *zip.File
	for _, file := range zipReader.File {
		if strings.HasSuffix(file.Name, "Prezzi.xml") {
			tariffFile = file
			break
		}
	}
	if tariffFile == nil {
		return nil, fmt.Errorf("tariff file not found in downloaded ZIP archive")
	}

	f, err := tariffFile.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Process the received data
	var dataSet NewDataSet
	if err := xml.NewDecoder(f).Decode(&dataSet); err != nil {
		return nil, err
	}

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

		location, err := time.LoadLocation("Europe/Rome")
		if err != nil {
			return nil, fmt.Errorf("load location: %w", err)
		}

		price, err := strconv.ParseFloat(strings.ReplaceAll(p.PUN, ",", "."), 64)
		if err != nil {
			return nil, fmt.Errorf("parse price: %w", err)
		}

		ts := time.Date(date.Year(), date.Month(), date.Day(), hour-1, 0, 0, 0, location)
		ar := api.Rate{
			Start: ts,
			End:   ts.Add(time.Hour),
			Value: t.totalPrice(price/1e3, ts),
		}
		data = append(data, ar)
	}

	data.Sort()
	return data, nil
}
