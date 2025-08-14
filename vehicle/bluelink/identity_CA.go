package bluelink

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

/* func (v *Identity) Login(user, password, language, region, brand string) (err error) {
	if user == "" || password == "" {
		return api.ErrMissingCredentials
	}
	// determine what login to use depending on `region`
	switch region {
	case RegionAustralia:
		err = v.LoginAU(user, password, language, brand)
	case RegionEurope:
		err = v.LoginEU(user, password, language, brand)
	case RegionCanada:
		err = v.LoginCA(user, password, language, brand)
	default:
		err = fmt.Errorf("unsupported region (%s)", region)
	}
	if err != nil {
		return fmt.Errorf("Login failed: %w", err)
	}

	return err
}
*/

func (v *Identity) LoginCA(user, password, language, brand string) (err error) {
	// hacking in the variables directly for now, move to config later
	var brandUrl string
	switch brand {
	case BrandGenesis:
		brandUrl = "genesisconnect.ca"
	case BrandHyundai:
		brandUrl = "mybluelink.ca"
	case BrandKia:
		brandUrl = "kiaconnect.ca"
	}

	apiUrl := fmt.Sprintf("https://%s/tods/api/", brandUrl)

	headers := map[string]string{
		"content-type":    "application/json",
		"accept":          "application/json",
		"accept-encoding": "gzip",
		"accept-language": "en.US,en;q=0.9",
		"host":            brandUrl,
		"client_id":       CAClientID,
		"client_secret":   CAClientSecret,
		"from":            "SPA",
		"language":        "0",
		"offset":          "-5",
		"sec-fetch-dest":  "empty",
		"sec-fetch-mode":  "cors",
		"sec-fetch-site":  "same-origin",
	}

	data := url.Values{
		"loginId":  {user},
		"password": {password},
	}

	// TODO: check whether this is used only here or if it can be moved
	// to the general headers
	headers["DeviceID"] = CADeviceID

	req, err := request.New(http.MethodPost, apiUrl, strings.NewReader(data.Encode()), headers)
	if err != nil {
		return err
	}

	var res map[string]any
	err = v.DoJSON(req, &res)
	if err != nil {
		return err
	}

	// extract values
	// since the JSON structure is unknown there's no other way than to map
	// through the return values :/
	var accessToken, refreshToken string
	var expiresIn int64

	if val, ok := res["result"]; ok {
		res = val.(map[string]any)
		if val, ok := res["token"]; ok {
			res = val.(map[string]any)
			if val, ok := res["accessToken"]; ok {
				accessToken = val.(string)
			} else {
				return fmt.Errorf("no access_token")
			}
			if val, ok := res["refreshToken"]; ok {
				refreshToken = val.(string)
			} else {
				return fmt.Errorf("no refresh_token")
			}
			if val, ok := res["expireIn"]; ok {
				if expiresIn, err = strconv.ParseInt(val.(string), 10, 64); err != nil {
					return fmt.Errorf("no expiresIn")
				}
			}
		} else {
			return fmt.Errorf("no token")
		}
	} else {
		return fmt.Errorf("no result")
	}

	// since we got here, all data was parsed successfully
	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn - 60, // make us request a refresh earlier than necessary
	}
	token = util.TokenWithExpiry(token)
	v.TokenSource = oauth.RefreshTokenSource(token, v)
	return err
}
