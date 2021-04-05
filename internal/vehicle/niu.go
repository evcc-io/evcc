package vehicle

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/vehicle/niu"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

const (
	niuAuth = "https://account-fk.niu.com"
	niuAPI  = "https://app-api-fk.niu.com"
)

// Niu is an api.Vehicle implementation for Niu vehicles
type Niu struct {
	*embed
	*request.Helper
	user, password, sn string
	tokens             niu.Token
	accessTokenExpiry  time.Time
	chargeStateG       func() (float64, error)
}

func init() {
	registry.Add("niu", NewNiuFromConfig)
}

// NewFordFromConfig creates a new vehicle
func NewNiuFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title              string
		Capacity           int64
		User, Password, SN string
		Cache              time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" || cc.SN == "" {
		return nil, errors.New("missing user, password or sn")
	}

	log := util.NewLogger("niu")

	v := &Niu{
		embed:    &embed{cc.Title, cc.Capacity},
		Helper:   request.NewHelper(log),
		user:     cc.User,
		password: cc.Password,
		sn:       strings.ToUpper(cc.SN),
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()

	return v, nil
}

// SoC implements the api.Vehicle interface
func (v *Niu) SoC() (float64, error) {
	return v.chargeStateG()
}

// chargeState implements the api.Vehicle interface
func (v *Niu) chargeState() (float64, error) {
	var socResp niu.SoC

	socReq, err := v.request(niuAPI + "/v3/motor_data/index_info?sn=" + v.sn)
	if err == nil {
		err = v.DoJSON(socReq, &socResp)
	}
	return float64(socResp.Data.Batteries.CompartmentA.BatteryCharging), err
}

// getAccessToken implements the Niu oauth2 api
func (v *Niu) getAccessToken() error {

	md5hash, err := getMD5Hash(v.password)
	if err != nil {
		return err
	}

	data := url.Values{
		"account":    []string{v.user},
		"password":   []string{md5hash},
		"grant_type": []string{"password"},
		"scope":      []string{"base"},
		"app_id":     []string{"niu_8xt1afu6"},
	}

	uri := niuAuth + "/v3/api/oauth2/token"
	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	})
	if err != nil {
		return err
	}

	var tokens niu.Token
	if err = v.DoJSON(req, &tokens); err == nil {
		v.tokens = tokens
		v.accessTokenExpiry = time.Unix(v.tokens.Data.Token.TokenExpiresIn, 0)
	}

	return err
}

// request implements the Niu web request
func (v *Niu) request(uri string) (*http.Request, error) {
	if v.tokens.Data.Token.AccessToken == "" || v.accessTokenExpiry.Before(time.Now()) {
		if err := v.getAccessToken(); err != nil {
			return nil, err
		}
	}

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"token": v.tokens.Data.Token.AccessToken,
	})

	return req, err
}

// getMD5Hash creates a MD5 hash based on a string
func getMD5Hash(text string) (string, error) {
	hasher := md5.New()
	if _, err := hasher.Write([]byte(text)); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
