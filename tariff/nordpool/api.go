package nordpool

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
)

func MakeURL(area string, date time.Time, currency string) (u *url.URL, err error) {
	u, err = url.Parse(BaseURL)
	q := u.Query()
	q.Add("market", "DayAhead")
	q.Add("deliveryArea", area)
	q.Add("currency", currency)
	q.Add("date", date.Format("2006-01-02"))
	u.RawQuery = q.Encode()
	return u, err
}

func (d *Date) UnmarshalJSON(b []byte) (err error) {
	loc, err := time.LoadLocation("CET")
	if err != nil {
		return err
	}

	date, err := time.ParseInLocation(`"2006-01-02"`, string(b), loc)
	if err != nil {
		return err
	}

	d.Date = date
	return nil
}

func GetDayAheadData(area string, date time.Time, currency string) (rc int, n *NordpoolResponse, err error) {
	url, err := MakeURL(area, date, currency)
	if err != nil {
		return 0, nil, err
	}

	resp, err := http.Get(url.String())

	if err != nil {
		return resp.StatusCode, nil, err
	}

	if resp.StatusCode == 204 {
		//fmt.Println("No data yet")
		return 204, nil, nil
	}

	b, err := io.ReadAll(resp.Body)

	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	var m NordpoolResponse
	err = json.Unmarshal(b, &m)
	return resp.StatusCode, &m, err
}
