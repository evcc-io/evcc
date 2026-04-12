package fritz

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/text/encoding/unicode"
)

// https://fritz.com/fileadmin/user_upload/Global/Service/Schnittstellen/AVM_Technical_Note_-_Session_ID_english_2021-05-03.pdf
const SessionTimeout = 15 * time.Minute

// FritzDECT settings
type Settings struct {
	URI, AIN, User, Password string
	Firmware82               bool // use new REST API (FritzOS 8.2+)
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
		if challresp, err = s.createChallengeResponse(v.Challenge); err == nil {
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

// createChallengeResponse creates the Fritzbox challenge response string
func (s Settings) createChallengeResponse(challenge string) (string, error) {
	encoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
	utf16le, err := encoder.String(challenge + "-" + s.Password)
	if err != nil {
		return "", err
	}

	hash := md5.Sum([]byte(utf16le))
	md5hash := hex.EncodeToString(hash[:])

	return challenge + "-" + md5hash, nil
}
