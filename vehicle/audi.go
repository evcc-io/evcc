package vehicle

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
)

const (
	audiURL        = "https://msg.audi.de/fs-car"
	audiDE         = "Audi/DE"
	audiAuthPrefix = "AudiAuth 1"
)

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
	Charger struct {
		Status struct {
			BatteryStatusData struct {
				StateOfCharge struct {
					Content int
				}
			}
		}
	}
}

// Audi is an api.Vehicle implementation for Audi cars
type Audi struct {
	*embed
	*util.HTTPHelper
	user, password, vin string
	token               string
	tokenValid          time.Time
	chargeStateG        provider.FloatGetter
}

// NewAudiFromConfig creates a new vehicle
func NewAudiFromConfig(log *util.Logger, other map[string]interface{}) api.Vehicle {
	cc := struct {
		Title               string
		Capacity            int64
		User, Password, VIN string
		Cache               time.Duration
	}{}
	util.DecodeOther(log, other, &cc)

	v := &Audi{
		embed:      &embed{cc.Title, cc.Capacity},
		HTTPHelper: util.NewHTTPHelper(util.NewLogger("audi")),
		user:       cc.User,
		password:   cc.Password,
		vin:        cc.VIN,
	}

	v.chargeStateG = provider.NewCached(log, v.chargeState, cc.Cache).FloatGetter()

	return v
}

func (v *Audi) apiURL(service, part string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", audiURL, service, "v1", audiDE, part)
}

func (v *Audi) headers(header *http.Header) {
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

func (v *Audi) login(user, password string) error {
	uri := v.apiURL("core/auth", "token")

	data := url.Values{
		"grant_type": []string{"password"},
		"username":   []string{user},
		"password":   []string{password},
	}

	req, err := http.NewRequest(http.MethodPost, uri, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	v.headers(&req.Header)
	req.Header.Set("Authorization", audiAuthPrefix)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var tr audiTokenResponse
	if b, err := v.RequestJSON(req, &tr); err != nil {
		if len(b) > 0 {
			var er audiErrorResponse
			if err = json.Unmarshal(b, &er); err == nil {
				return errors.New(er.Description)
			}
		}
		return err
	}

	v.token = tr.AccessToken
	v.tokenValid = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)

	return nil
}

func (v *Audi) request(uri string) (*http.Request, error) {
	if v.token == "" || time.Since(v.tokenValid) > 0 {
		if err := v.login(v.user, v.password); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return req, err
	}

	v.headers(&req.Header)
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", audiAuthPrefix, v.token))

	return req, nil
}

// chargeState implements the Vehicle.ChargeState interface
func (v *Audi) chargeState() (float64, error) {
	uri := v.apiURL("bs/batterycharge", fmt.Sprintf("vehicles/%s/charger", v.vin))
	req, err := v.request(uri)
	if err != nil {
		return 0, err
	}

	var br audiBatteryResponse
	_, err = v.RequestJSON(req, &br)

	return float64(br.Charger.Status.BatteryStatusData.StateOfCharge.Content), err
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Audi) ChargeState() (float64, error) {
	return v.chargeStateG()
}
