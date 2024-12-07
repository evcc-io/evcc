package nordpool

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
)

func MakeURL(area string, date time.Time, currency string) string {
	data := url.Values{
		"market":       {"DayAhead"},
		"deliveryArea": {area},
		"currency":     {currency},
		"date":         {date.Format("2006-01-02")},
	}

	return BaseURL + "?" + data.Encode()
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
	uri := MakeURL(area, date, currency)

	resp, err := http.Get(uri)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	if resp.StatusCode == 204 {
		// fmt.Println("No data yet")
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
