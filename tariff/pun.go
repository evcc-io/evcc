package tariff

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
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
	"golang.org/x/net/html"
)

type Pun struct {
	*embed
	log     *util.Logger
	charges float32
	tax     float32
	data    *util.Monitor[api.Rates]
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
	cc := struct {
		Charges float32
		Tax     float32
	}{
		Charges: 0.0,
		Tax:     0.0,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	t := &Pun{
		log:     util.NewLogger("pun"),
		charges: cc.Charges,
		tax:     cc.Tax,
		data:    util.NewMonitor[api.Rates](2 * time.Hour),
	}

	done := make(chan error)
	go t.run(done)
	err := <-done

	return t, err
}

func (t *Pun) run(done chan error) {
	var once sync.Once
	bo := newBackoff()

	tick := time.NewTicker(time.Hour)
	for ; true; <-tick.C {
		var today api.Rates
		if err := backoff.Retry(func() error {
			var err error

			today, err = t.getData(time.Now())

			return err
		}, bo); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		var tomorrow api.Rates
		if err := backoff.Retry(func() error {
			var err error

			tomorrow, err = t.getData(time.Now().AddDate(0, 0, 1))

			return err
		}, bo); err != nil {
			once.Do(func() { done <- err })
			t.log.ERROR.Println(err)
			continue
		}

		// merge today and tomorrow data
		data := append(today, tomorrow...)
		t.data.Set(data)

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
	// Initial URL
	urlString := "https://www.mercatoelettrico.org/It/WebServerDataStore/MGP_Prezzi/" + day.Format("20060102") + "MGPPrezzi.xml"

	// Cookie Jar zur Speicherung von Cookies zwischen den Requests
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
	}

	// Erster Request
	resp, err := client.Get(urlString)
	if err != nil {
		fmt.Println("Error fetching URL:", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	formData, err := parseFormFields(body)
	if err != nil {
		fmt.Println("Error parsing form fields:", err)
		return nil, err
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
	resp, err = client.Get(urlString)
	if err != nil {
		fmt.Println("Error fetching URL after form submission:", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Verarbeitung der erhaltenen Daten
	body, _ = io.ReadAll(resp.Body)
	xmlData := []byte(string(body)) // Ersetzen Sie [Ihr XML-Datenstring hier] mit Ihrem XML-String

	var dataSet NewDataSet
	err = xml.Unmarshal(xmlData, &dataSet)
	if err != nil {
		fmt.Println("Error unmarshalling XML: ", err)
	}

	data := make(api.Rates, 0, len(dataSet.Prezzi))

	for _, p := range dataSet.Prezzi {
		date, err := time.Parse("20060102", p.Data)
		if err != nil {
			fmt.Println("Error parsing date: ", err)
		}

		hour, err := strconv.Atoi(p.Ora)
		if err != nil {
			fmt.Println("Error parsing hour: ", err)
		}

		location, err := time.LoadLocation("Europe/Rome")
		if err != nil {
			fmt.Println("Error loading location: ", err)
		}

		start := time.Date(date.Year(), date.Month(), date.Day(), hour-1, 0, 0, 0, location)
		end := start.Add(time.Hour)

		priceStr := strings.Replace(p.PUN, ",", ".", -1) // Ersetzen Sie Komma durch Punkt
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			fmt.Println("Error parsing price: ", err)
		}

		ar := api.Rate{
			Start: start,
			End:   end,
			Price: (price/1000.0 + float64(t.charges)) * (1 + float64(t.tax)),
		}
		data = append(data, ar)
	}

	data.Sort()
	return data, nil
}

func parseFormFields(body []byte) (url.Values, error) {
	formData := url.Values{}
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return formData, err
	}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "input" {
			inputType := ""
			inputName := ""
			inputValue := ""
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
				formData.Set(inputName, inputValue)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return formData, nil
}
