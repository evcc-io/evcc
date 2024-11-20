package tariff

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http/cookiejar"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/net/html"
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

func NewPunFromConfig(other map[string]interface{}) (api.Tariff, error) {
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

	done := make(chan error)
	go t.run(done)
	err := <-done

	return t, err
}

func (t *Pun) run(done chan error) {
	var once sync.Once

	tick := time.NewTicker(time.Hour)
	for ; true; <-tick.C {
		var today api.Rates
		if err := backoff.Retry(func() error {
			var err error

			today, err = t.getData(time.Now())

			return err
		}, bo()); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

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
	// Cookie Jar zur Speicherung von Cookies zwischen den Requests
	client := request.NewClient(t.log)
	client.Jar, _ = cookiejar.New(nil)

	// Erster Request
	uri := "https://storico.mercatoelettrico.org/It/WebServerDataStore/MGP_Prezzi/" + day.Format("20060102") + "MGPPrezzi.xml"
	resp, err := client.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	formData, err := parseFormFields(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("form fields: %w", err)
	}

	redirectURL := resp.Request.URL.String()

	// Hinzufügen der spezifizierten Parameter
	formData.Set("ctl00$ContentPlaceHolder1$CBAccetto1", "on")
	formData.Set("ctl00$ContentPlaceHolder1$CBAccetto2", "on")
	formData.Set("ctl00$ContentPlaceHolder1$Button1", "Accetto")

	// Formular senden
	resp, err = client.PostForm(redirectURL, formData)
	if err != nil {
		fmt.Println("Error submitting form:", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Erneuter Request auf die ursprüngliche URL
	resp, err = client.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Verarbeitung der erhaltenen Daten
	var dataSet NewDataSet
	if err := xml.NewDecoder(resp.Body).Decode(&dataSet); err != nil {
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

		location, err := time.LoadLocation("Europe/Rome")
		if err != nil {
			return nil, fmt.Errorf("load location: %w", err)
		}

		start := time.Date(date.Year(), date.Month(), date.Day(), hour-1, 0, 0, 0, location)
		end := start.Add(time.Hour)

		priceStr := strings.Replace(p.PUN, ",", ".", -1) // Ersetzen Sie Komma durch Punkt
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			return nil, fmt.Errorf("parse price: %w", err)
		}

		ar := api.Rate{
			Start: start,
			End:   end,
			Price: t.totalPrice(price / 1e3),
		}
		data = append(data, ar)
	}

	data.Sort()
	return data, nil
}

func parseFormFields(body io.Reader) (url.Values, error) {
	data := url.Values{}
	doc, err := html.Parse(body)
	if err != nil {
		return data, err
	}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "input" {
			var inputType, inputName, inputValue string
			for _, a := range n.Attr {
				if a.Key == "type" {
					inputType = a.Val
				} else if a.Key == "name" {
					inputName = a.Val
				} else if a.Key == "value" {
					inputValue = a.Val
				}
			}
			if inputType == "hidden" && inputName != "" {
				data.Set(inputName, inputValue)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return data, nil
}
