package fritzbox

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/text/encoding/unicode"
)

// AVM FritzBox AHA interface and authentification specifications:
// https://avm.de/fileadmin/user_upload/Global/Service/Schnittstellen/AHA-HTTP-Interface.pdf
// https://avm.de/fileadmin/user_upload/Global/Service/Schnittstellen/AVM_Technical_Note_-_Session_ID.pdf

// FritzBox connection
type Connection struct {
	*request.Helper
	URI, AIN, User, Password, SID string
	Updated                       time.Time
}

func (fb *Connection) ExecFritzDectCmd(function string) (string, error) {
	// refresh Fritzbox session id
	if time.Since(fb.Updated).Minutes() >= 10 {
		err := fb.getSessionID()
		if err != nil {
			return "", err
		}
		// update session timestamp
		fb.Updated = time.Now()
	}

	parameters := url.Values{
		"sid":       []string{fb.SID},
		"ain":       []string{fb.AIN},
		"switchcmd": []string{function},
	}

	uri := fmt.Sprintf("%s/webservices/homeautoswitch.lua", fb.URI)
	response, err := fb.GetBody(uri + "?" + parameters.Encode())
	return strings.TrimSpace(string(response)), err
}

// CurrentPower implements the api interface
func (fb *Connection) CurrentPower() (float64, error) {
	// power value in 0,001 W (current switch power, refresh approximately every 2 minutes)
	resp, err := fb.ExecFritzDectCmd("getswitchpower")
	if err != nil {
		return 0, err
	}

	if resp == "inval" {
		return 0, api.ErrNotAvailable
	}
	power, err := strconv.ParseFloat(resp, 64)
	power = power / 1000 // mW ==> W

	return power, err
}

// Fritzbox helpers (credits to https://github.com/rsdk/ahago)

// getSessionID fetches a session-id based on the username and password in the connection struct
func (fb *Connection) getSessionID() error {
	uri := fmt.Sprintf("%s/login_sid.lua", fb.URI)
	body, err := fb.GetBody(uri)
	if err != nil {
		return err
	}

	v := struct {
		SID       string
		Challenge string
		BlockTime string
	}{}

	if err = xml.Unmarshal(body, &v); err == nil && v.SID == "0000000000000000" {
		var challresp string
		if challresp, err = createChallengeResponse(v.Challenge, fb.Password); err == nil {
			params := url.Values{
				"username": []string{fb.User},
				"response": []string{challresp},
			}

			if body, err = fb.GetBody(uri + "?" + params.Encode()); err == nil {
				err = xml.Unmarshal(body, &v)
				if v.SID == "0000000000000000" {
					return errors.New("invalid username (" + fb.User + ") or password")
				}
				fb.SID = v.SID
			}
		}
	}

	return err
}

// createChallengeResponse creates the Fritzbox challenge response string
func createChallengeResponse(challenge string, pass string) (string, error) {
	encoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
	utf16le, err := encoder.String(challenge + "-" + pass)
	if err != nil {
		return "", err
	}

	hash := md5.New()
	if _, err = hash.Write([]byte(utf16le)); err != nil {
		return "", err
	}

	md5hash := hex.EncodeToString(hash.Sum(nil))
	return challenge + "-" + md5hash, nil
}
