package bluelink

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
)

func PopulateSettingsAU(brand, region string) (BluelinkConfig, error) {
	// sru_250814: I know this looks weird but right now it is unclear whether all
	// 	regional bluelink version use at least a roughly similar structure so I use
	//	a map for now.
	return BluelinkConfig{
		URI:               ConfigMap[region][brand]["URI"],
		BasicToken:        ConfigMap[region][brand]["BasicToken"],
		CCSPServiceID:     ConfigMap[region][brand]["ServiceId"],
		CCSPApplicationID: ConfigMap[region][brand]["AppId"],
		AuthClientID:      ConfigMap[region][brand]["AuthClientId"],
		BrandAuthUrl:      ConfigMap[region][brand]["BrandAuthUrl"],
		PushType:          ConfigMap[region][brand]["PushType"],
		Cfb:               ConfigMap[region][brand]["Cfb"],
		LoginFormHost:     ConfigMap[region][brand]["LoginFormHost"],
	}, nil
}

func (v *Identity) LoginAU(user, password string) (err error) {
	v.deviceID, err = v.getDeviceID()
	if err != nil {
		return err
	}

	cookieClient, err := v.getCookies()
	if err != nil {
		return err
	}

	code, err := v.brandLoginAU(cookieClient, user, password)
	if err != nil {
		return err
	}

	if token, err := v.exchangeCodeHyundai(code); err == nil {
		v.TokenSource = oauth.RefreshTokenSource(token, v)
	}

	return err
}

func (v *Identity) brandLoginAU(cookieClient *request.Helper, user, password string) (string, error) {
	data := map[string]string{
		"email":    user,
		"password": password,
	}

	// try to get the redirect URL
	cookieClient.CheckRedirect = request.DontFollow

	var res struct {
		RedirectUrl string `json:"redirectUrl"`
	}

	req, err := request.New(http.MethodPost, v.config.URI+LoginURL, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return "", err
	}

	err = cookieClient.DoJSON(req, &res)
	if err != nil {
		return "", err
	}

	uri, err := url.Parse(res.RedirectUrl)
	if err != nil {
		return "", err
	}

	if code := uri.Query().Get("code"); len(code) > 0 {
		return code, nil
	}

	return "", errors.New("authorization code not found")
}
