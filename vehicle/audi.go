package vehicle

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"golang.org/x/net/publicsuffix"
)

const (
	audiURL        = "https://msg.audi.de/fs-car"
	audiDE         = "Audi/DE"
	audiAuthPrefix = "AudiAuth 1"
)

type audiTokenResponse struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
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
	chargeStateG        func() (float64, error)
}

func init() {
	registry.Add("audi", NewAudiFromConfig)
}

// NewAudiFromConfig creates a new vehicle
func NewAudiFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title               string
		Capacity            int64
		User, Password, VIN string
		Cache               time.Duration
	}{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &Audi{
		embed:      &embed{cc.Title, cc.Capacity},
		HTTPHelper: util.NewHTTPHelper(util.NewLogger("audi")),
		user:       cc.User,
		password:   cc.Password,
		vin:        cc.VIN,
	}

	// v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()

	var err error
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		panic(err)
	}

	v.Client = &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	println("1.")

	var uri, ref string
	var vars formVars
	var req *http.Request
	var resp *http.Response
	// var dump []byte

	// var res interface{}
	// uri := "https://app-api.live-my.audi.com/myaudiappidk/v1/openid-configuration"
	// req, err := http.NewRequest(http.MethodGet, uri, nil)
	// dump, _ := httputil.DumpRequest(req, true)
	// fmt.Printf("%s\n", dump)
	// _, _ = v.RequestJSON(req, &res)
	// dump, _ = httputil.DumpResponse(v.LastResponse(), true)
	// fmt.Printf("%s\n\n", dump)

	println("2.")

	uri = "https://identity.vwgroup.io/oidc/v1/authorize?response_type=code&client_id=09b6cbec-cd19-4589-82fd-363dfa8c24da%40apps_vw-dilab_com&redirect_uri=myaudi%3A%2F%2F%2F&scope=address%20profile%20badge%20birthdate%20birthplace%20nationalIdentifier%20nationality%20profession%20email%20vin%20phone%20nickname%20name%20picture%20mbb%20gallery%20openid&state=7f8260b5-682f-4db8-b171-50a5189a1c08&nonce=583b9af2-7799-4c72-9cb0-e6c0f42b87b3&prompt=login&ui_locales=de-DE"
	resp, err = v.Client.Get(uri)
	if err != nil {
		panic(err)
	}

	println("3.")

	uri = resp.Header.Get("Location")
	resp, err = v.Client.Get(uri)
	if err != nil {
		panic(err)
	}

	vars, err = formValues(resp.Body, "form#emailPasswordForm")
	if err != nil {
		panic(err)
	}

	println("4.1.")

	ref = uri
	uri = "https://identity.vwgroup.io" + vars.action
	body := fmt.Sprintf("_csrf=%s&relayState=%s&hmac=%s&email=%s", vars.csrf, vars.relayState, vars.hmac, url.QueryEscape(v.user))
	req, err = http.NewRequest(http.MethodPost, uri, strings.NewReader(body))
	if err != nil {
		panic(err)
	}
	req.Header.Add("Referer", ref)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err = v.Client.Do(req)
	if err != nil {
		panic(err)
	}

	println("4.2.")

	ref = uri
	uri = "https://identity.vwgroup.io" + resp.Header.Get("Location")
	req, err = http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		panic(err)
	}
	resp, err = v.Client.Do(req)
	if err != nil {
		panic(err)
	}

	vars, err = formValues(resp.Body, "form#credentialsForm")
	if err != nil {
		panic(err)
	}

	println("5.")

	uri = "https://identity.vwgroup.io" + vars.action
	body = fmt.Sprintf("_csrf=%s&relayState=%s&email=%s&hmac=%s&password=%s",
		vars.csrf,
		vars.relayState,
		url.QueryEscape(v.user),
		vars.hmac,
		url.QueryEscape(v.password),
	)
	req, err = http.NewRequest(http.MethodPost, uri, strings.NewReader(body))
	if err != nil {
		panic(err)
	}
	req.Header.Add("Referer", ref)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err = v.Client.Do(req)
	if err != nil {
		panic(err)
	}

	println("6.")

	uri = resp.Header.Get("Location")
	resp, err = v.Client.Get(uri)
	if err != nil {
		panic(err)
	}

	println("7.")

	uri = resp.Header.Get("Location")
	resp, err = v.Client.Get(uri)
	if err != nil {
		panic(err)
	}

	println("8.")

	uri = resp.Header.Get("Location")
	resp, err = v.Client.Get(uri)
	if err != nil {
		panic(err)
	}

	location, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		panic(err)
	}
	code := location.Query().Get("code")

	println("9.")

	uri = "https://app-api.my.audi.com/myaudiappidk/v1/token"
	body = fmt.Sprintf("client_id=%s&grant_type=%s&code=%s&redirect_uri=%s&response_type=%s",
		url.QueryEscape("09b6cbec-cd19-4589-82fd-363dfa8c24da@apps_vw-dilab_com"),
		"authorization_code",
		code,
		url.QueryEscape("myaudi:///"),
		url.QueryEscape("token id_token"),
	)
	req, err = http.NewRequest(http.MethodPost, uri, strings.NewReader(body))
	if err != nil {
		panic(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err = v.Client.Do(req)
	if err != nil {
		panic(err)
	}
	b, err := ioutil.ReadAll(resp.Body)
	var tokens audiTokenResponse
	err = json.Unmarshal(b, &tokens)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n\n", tokens)

	println("10.")

	uri = "https://mbboauth-1d.prd.ece.vwg-connect.com/mbbcoauth/mobile/oauth2/v1/token"
	body = fmt.Sprintf("grant_type=%s&token=%s&scope=%s", "id_token", tokens.IDToken, "sc2:fal")
	req, err = http.NewRequest(http.MethodPost, uri, strings.NewReader(body))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("X-App-Version", "3.14.0")
	req.Header.Add("X-App-Name", "myAudi")
	req.Header.Add("X-Client-Id", "77869e21-e30a-4a92-b016-48ab7d3db1d8")
	resp, err = v.Client.Do(req)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(b, &tokens)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", tokens)

	return v, nil
}

type formVars struct {
	action     string
	csrf       string
	relayState string
	hmac       string
}

func formValues(reader io.Reader, id string) (formVars, error) {
	vars := formVars{}

	doc, err := goquery.NewDocumentFromReader(reader)
	if err == nil {
		form := doc.Find(id).First()
		if form.Length() != 1 {
			return vars, errors.New("unexpected length")
		}

		var exists bool
		vars.action, exists = form.Attr("action")
		if !exists {
			return vars, errors.New("attribute not found")
		}

		vars.csrf, err = attr(form, "input[name=_csrf]", "value")
		if err == nil {
			vars.relayState, err = attr(form, "input[name=relayState]", "value")
		}
		if err == nil {
			vars.hmac, err = attr(form, "input[name=hmac]", "value")
		}
	}

	return vars, err
}

func attr(doc *goquery.Selection, path, attr string) (res string, err error) {
	sel := doc.Find(path)
	if sel.Length() != 1 {
		return "", errors.New("unexpected length")
	}

	v, exists := sel.Attr(attr)
	if !exists {
		return "", errors.New("attribute not found")
	}

	return v, nil
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Audi) ChargeState() (float64, error) {
	// return v.chargeStateG()
	return 0, nil
}
