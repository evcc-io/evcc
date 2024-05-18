package nissan

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

const (
	APIVersion   = "protocol=1.0,resource=2.1"
	ClientID     = "a-ncb-nc-android-prod"
	ClientSecret = "6GKIax7fGT5yPHuNmWNVOc4q5POBw1WRSW39ubRA8WPBmQ7MOxhm75EsmKMKENem"
	Scope        = "openid profile vehicles"
	Realm        = "a-ncb-prod"
	AuthURL      = "https://prod.eu2.auth.kamereon.org/kauth"
	RedirectURI  = "org.kamereon.service.nci:/oauth2redirect"
)

type Identity struct {
	*request.Helper
	oauth2.TokenSource
}

// NewIdentity creates Nissan identity
func NewIdentity(log *util.Logger) *Identity {
	return &Identity{
		Helper: request.NewHelper(log),
	}
}

func (v *Identity) Login(user, password string) error {
	uri := fmt.Sprintf("%s/json/realms/root/realms/%s/authenticate", AuthURL, Realm)
	req, err := request.New(http.MethodPost, uri, nil, map[string]string{
		"Accept-Api-Version": APIVersion,
		"X-Username":         "anonymous",
		"X-Password":         "anonymous",
		"Accept":             "application/json",
	})

	var nToken Token
	var realm string
	var code string

	if err == nil {
		var res Auth
		if err = v.DoJSON(req, &res); err != nil {
			return err
		}

		for id, cb := range res.Callbacks {
			switch cb.Type {
			case "NameCallback":
				res.Callbacks[id].Input[0].Value = user
			case "PasswordCallback":
				res.Callbacks[id].Input[0].Value = password
			}
		}

		// https://github.com/Tobiaswk/dartnissanconnect/commit/7d28dd5461aaed3e46b5be0c9fd58887e1e0cd0b
		err = api.ErrNotAvailable // not nil
		for attempt := 1; attempt <= 10 && err != nil; attempt++ {
			req, err = request.New(http.MethodPost, uri, request.MarshalJSON(res), map[string]string{
				"Accept-Api-Version": APIVersion,
				"X-Username":         "anonymous",
				"X-Password":         "anonymous",
				"Content-type":       "application/json",
				"Accept":             "application/json",
			})
			if err == nil {
				if err = v.DoJSON(req, &nToken); err != nil && !nToken.SessionExpired() {
					break
				}
			}
		}

		if errT := nToken.Error(); err != nil && errT != nil {
			err = errT
		}

		realm = strings.Trim(nToken.Realm, "/")
	}

	if err == nil {
		data := url.Values{
			"client_id":     {ClientID},
			"redirect_uri":  {RedirectURI},
			"response_type": {"code"},
			"scope":         {Scope},
			"nonce":         {"sdfdsfez"},
		}

		uri := fmt.Sprintf("%s/oauth2/%s/authorize?%s", AuthURL, realm, data.Encode())
		req, err = request.New(http.MethodGet, uri, nil, map[string]string{
			"Cookie": "i18next=en-UK; amlbcookie=05; kauthSession=" + nToken.TokenID,
		})

		if err == nil {
			var param request.InterceptResult
			v.Client.CheckRedirect, param = request.InterceptRedirect("code", true)

			if _, err = v.Do(req); err == nil {
				code, err = param()
			}

			v.CheckRedirect = nil
		}
	}

	uri = fmt.Sprintf("%s/oauth2/%s/access_token", AuthURL, realm)
	config := oauth2.Config{
		ClientID:     ClientID,
		ClientSecret: ClientSecret,
		RedirectURL:  RedirectURI,
		Endpoint: oauth2.Endpoint{
			AuthURL:   uri,
			TokenURL:  uri,
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, v.Client)

	var token *oauth2.Token
	if err == nil {
		token, err = config.Exchange(ctx, code)
	}

	v.TokenSource = config.TokenSource(ctx, token)

	return err
}
