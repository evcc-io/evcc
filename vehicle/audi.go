package vehicle

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
)

const (
	audiURL         = "https://msg.audi.de/fs-car"
	audiDE          = "Audi/DE"
	audiAuthPrefix  = "AudiAuth 1"
	audiValidMargin = 10 * time.Second
)

// Audi is an api.Vehicle implementation for Audi cars
type Audi struct {
	*embed
	user, password, vin string
	token               string
	tokenValid          time.Time
	cache               time.Duration
	chargeStateVal      float64
	chargeStateUpdated  time.Time
}

type audiTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}
type audiErrorResponse struct {
	Error       string
	Description string `json:"error_description"`
}

type audiBatteryResponse struct {
	Charger audiBrCharger
}

type audiBrCharger struct {
	Status audiBrStatus
}

type audiBrStatus struct {
	BatteryStatusData audiBrStatusData
}

type audiBrStatusData struct {
	StateOfCharge audiBrStateOfCharge
}

type audiBrStateOfCharge struct {
	Content int
}

// NewAudiFromConfig creates a new vehicle
func NewAudiFromConfig(log *api.Logger, other map[string]interface{}) api.Vehicle {
	cc := struct {
		Title               string
		Capacity            int64
		User, Password, VIN string
		Cache               time.Duration
	}{}
	api.DecodeOther(log, other, &cc)

	return &Audi{
		embed:    &embed{cc.Title, cc.Capacity},
		user:     cc.User,
		password: cc.Password,
		vin:      cc.VIN,
		cache:    cc.Cache,
	}
}

func (m *Audi) apiURL(service, part string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", audiURL, service, "v1", audiDE, part)
}

func (m *Audi) headers(header *http.Header) {
	for k, v := range map[string]string{
		"Accept":        "application/json",
		"X-App-ID":      "de.audi.mmiapp",
		"X-App-Name":    "MMIconnect",
		"X-App-Version": "2.8.3",
		"X-Brand":       "audi",
		"X-Country-Id":  "DE",
		"X-Language-Id": "de",
		"X-Platform":    "google",
		"User-Agent":    "okhttp/2.7.4",
		"ADRUM_1":       "isModule:true",
		"ADRUM":         "isAray:true",
	} {
		header.Set(k, v)
	}
}

func (m *Audi) login(user, password string) error {
	uri := m.apiURL("core/auth", "token")

	data := url.Values{
		"grant_type": []string{"password"},
		"username":   []string{user},
		"password":   []string{password},
	}

	req, err := http.NewRequest(http.MethodPost, uri, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	m.headers(&req.Header)
	req.Header.Set("Authorization", audiAuthPrefix)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		var er audiErrorResponse
		if err = json.Unmarshal(b, &er); err == nil {
			return errors.New(er.Description)
		}
		return fmt.Errorf("unexpected response %d: %s", resp.StatusCode, string(b))
	}

	var tr audiTokenResponse
	if err = json.Unmarshal(b, &tr); err != nil {
		return err
	}

	m.token = tr.AccessToken
	m.tokenValid = time.Now().Add(time.Duration(tr.ExpiresIn)*time.Second - audiValidMargin)

	return nil
}

func (m *Audi) request(uri string) (*http.Request, error) {
	// token invalid or expired
	if m.token == "" || time.Now().After(m.tokenValid) {
		if err := m.login(m.user, m.password); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return req, err
	}

	m.headers(&req.Header)
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", audiAuthPrefix, m.token))

	return req, nil
}

// chargeState implements the SoC.ChargeState interface internally
func (m *Audi) chargeState() (float64, error) {
	uri := m.apiURL("bs/batterycharge", fmt.Sprintf("vehicles/%s/charger", m.vin))
	req, err := m.request(uri)
	if err != nil {
		return 0, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var br audiBatteryResponse
	err = json.Unmarshal(b, &br)

	return float64(br.Charger.Status.BatteryStatusData.StateOfCharge.Content), err
}

// ChargeState implements the SoC.ChargeState interface
func (m *Audi) ChargeState() (float64, error) {
	var err error
	if time.Since(m.chargeStateUpdated) > m.cache {
		m.chargeStateVal, err = m.chargeState()
		if err == nil {
			m.chargeStateUpdated = time.Now()
		}
	}

	return m.chargeStateVal, err
}
