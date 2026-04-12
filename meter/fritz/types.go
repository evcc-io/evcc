package fritz

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"

	"github.com/evcc-io/evcc/util/request"
)

// FritzDECT settings
type Settings struct {
	URI, AIN, User, Password string
	Legacy                   bool // use legacy homeautoswitch.lua API
}

// Fritzbox helpers (credits to https://github.com/rsdk/ahago)

// getSessionID fetches a session-id based on the username and password in the connection struct
func (s Settings) GetSessionID(c *request.Helper) (string, error) {
	uri := fmt.Sprintf("%s/login_sid.lua", s.URI)
	body, err := c.GetBody(uri)
	if err != nil {
		return "", err
	}

	var v struct {
		SID       string
		Challenge string
	}

	if err = xml.Unmarshal(body, &v); err == nil && v.SID == "0000000000000000" {
		var challresp string
		if challresp, err = CreateChallengeResponse(v.Challenge, s.Password); err == nil {
			params := url.Values{
				"username": {s.User},
				"response": {challresp},
			}

			if body, err = c.GetBody(uri + "?" + params.Encode()); err == nil {
				if err = xml.Unmarshal(body, &v); err == nil && v.SID == "0000000000000000" {
					return "", errors.New("invalid user or password")
				}
			}
		}
	}

	return v.SID, err
}
