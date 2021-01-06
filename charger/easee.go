package charger

import (
	"fmt"
	"net/http"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

// https://api.easee.cloud/index.html

const easyAPI = "https://easee.cloud/api"

// EaseeToken is the /api/accounts/token and /api/accounts/refresh_token response
type EaseeToken struct {
	AccessToken  string    `json:"accessToken"`
	ExpiresIn    int       `json:"expiresIn"`
	TokenType    string    `json:"tokenType"`
	RefreshToken string    `json:"refreshToken"`
	Valid        time.Time // helper to store validity timestamp
}

// Easee charger implementation
type Easee struct {
	*request.Helper
	token EaseeToken
}

func init() {
	registry.Add("easee", NewEaseeFromConfig)
}

// NewEaseeFromConfig creates a go-e charger from generic config
func NewEaseeFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		User     string
		Password string
		Charger  string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEasee(cc.User, cc.Password, cc.Charger)
}

// NewEasee creates Easee charger
func NewEasee(user, password, charger string) (*Easee, error) {
	c := &Easee{
		Helper: request.NewHelper(util.NewLogger("easee")),
	}

	err := c.login(user, password)

	return c, err
}

func (c *Easee) login(user, password string) error {
	data := struct {
		Username string `json:"userName"`
		Password string `json:"password"`
	}{
		Username: user,
		Password: password,
	}

	uri := fmt.Sprintf("%s/%s", easyAPI, "/accounts/token")
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

	if err == nil {
		err = c.DoJSON(req, &c.token)
		c.token.Valid = time.Now().Add(time.Second * time.Duration(c.token.ExpiresIn))
	}

	return err
}

func (c *Easee) refreshToken() error {
	data := struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}{
		AccessToken:  c.token.AccessToken,
		RefreshToken: c.token.RefreshToken,
	}

	uri := fmt.Sprintf("%s/%s", easyAPI, "/accounts/refresh_token")
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

	var token EaseeToken
	if err == nil {
		err = c.DoJSON(req, &token)
		token.Valid = time.Now().Add(time.Second * time.Duration(token.ExpiresIn))
	}

	if err == nil {
		c.token = token
	}

	return err
}

// Status implements the Charger.Status interface
func (c *Easee) Status() (api.ChargeStatus, error) {
	status, err := c.apiStatus()
	if err != nil {
		return api.StatusNone, err
	}

	switch status.Car {
	case 1:
		return api.StatusA, nil
	case 2:
		return api.StatusC, nil
	case 3, 4:
		return api.StatusB, nil
	default:
		return api.StatusNone, fmt.Errorf("car unknown result: %d", status.Car)
	}
}

// Enabled implements the Charger.Enabled interface
func (c *Easee) Enabled() (bool, error) {
	status, err := c.apiStatus()
	if err != nil {
		return false, err
	}

	switch status.Alw {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, fmt.Errorf("alw unknown result: %d", status.Alw)
	}
}

// Enable implements the Charger.Enable interface
func (c *Easee) Enable(enable bool) error {
	var b int
	if enable {
		b = 1
	}

	status, err := c.apiUpdate(fmt.Sprintf("alw=%d", b))
	if err == nil && isValid(status) && status.Alw != b {
		return fmt.Errorf("alw update failed: %d", status.Amp)
	}

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (c *Easee) MaxCurrent(current int64) error {
	status, err := c.apiUpdate(fmt.Sprintf("amx=%d", current))
	if err == nil && isValid(status) && int64(status.Amp) != current {
		return fmt.Errorf("amp update failed: %d", status.Amp)
	}

	return err
}

// CurrentPower implements the Meter interface.
func (c *Easee) CurrentPower() (float64, error) {
	status, err := c.apiStatus()
	var power float64
	if len(status.Nrg) == 16 {
		power = float64(status.Nrg[11]) * 10
	}
	return power, err
}

// ChargedEnergy implements the ChargeRater interface
func (c *Easee) ChargedEnergy() (float64, error) {
	status, err := c.apiStatus()
	energy := float64(status.Dws) / 3.6e5 // Deka-Watt-Seconds to kWh (100.000 == 0,277kWh)
	return energy, err
}

// Currents implements the MeterCurrent interface
func (c *Easee) Currents() (float64, float64, float64, error) {
	status, err := c.apiStatus()
	if len(status.Nrg) == 16 {
		return float64(status.Nrg[4]) / 10, float64(status.Nrg[5]) / 10, float64(status.Nrg[6]) / 10, nil
	}
	return 0, 0, 0, err
}
