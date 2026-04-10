package fritzdect_new

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/text/encoding/unicode"
)

// NewConnection creates FritzDECT connection
func NewConnection(uri, ain, user, password string) (*Connection, error) {
	if uri == "" {
		uri = "https://fritz.box"
	}

	if ain == "" {
		return nil, errors.New("missing ain")
	}

	settings := &Settings{
		URI:      strings.TrimRight(uri, "/"),
		AIN:      ain,
		User:     user,
		Password: password,
	}

	log := util.NewLogger("fritzdect").Redact(password)

	fritzdect_new := &Connection{
		Helper:   request.NewHelper(log),
		Settings: settings,
	}

	fritzdect_new.Client.Transport = request.NewTripper(log, transport.Insecure())

	return fritzdect_new, nil
}

// ExecCmd execautes an FritzDECT AHA-HTTP-Interface command
func (c *Connection) ExecCmd(function string) (string, error) {
	// refresh Fritzbox session id
	if time.Since(c.updated) >= sessionTimeout {
		if err := c.getSessionID(); err != nil {
			return "", err
		}
		// update session timestamp
		c.updated = time.Now()
	}

	parameters := url.Values{
		"sid":       {c.SID},
		"ain":       {c.AIN},
		"switchcmd": {function},
	}

	uri := fmt.Sprintf("%s/webservices/homeautoswitch.lua", c.URI)
	body, err := c.GetBody(uri + "?" + parameters.Encode())

	res := strings.TrimSpace(string(body))

	if err == nil && res == "inval" {
		err = api.ErrNotAvailable
	}

	return res, err
}

// Fritzbox helpers (credits to https://github.com/rsdk/ahago)

// getSessionID fetches a session-id based on the username and password in the connection struct
func (c *Connection) getSessionID() error {
	uri := fmt.Sprintf("%s/login_sid.lua", c.URI)
	body, err := c.GetBody(uri)
	if err != nil {
		return err
	}

	var v struct {
		SID       string
		Challenge string
		BlockTime string
	}

	if err = xml.Unmarshal(body, &v); err == nil && v.SID == "0000000000000000" {
		var challresp string
		if challresp, err = createChallengeResponse(v.Challenge, c.Password); err == nil {
			params := url.Values{
				"username": {c.User},
				"response": {challresp},
			}

			if body, err = c.GetBody(uri + "?" + params.Encode()); err == nil {
				err = xml.Unmarshal(body, &v)
				if v.SID == "0000000000000000" {
					return errors.New("invalid user or password")
				}
				c.SID = v.SID
			}
		}
	}

	return err
}

// createChallengeResponse creates the Fritzbox challenge response string
func createChallengeResponse(challenge, pass string) (string, error) {
	encoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
	utf16le, err := encoder.String(challenge + "-" + pass)
	if err != nil {
		return "", err
	}

	hash := md5.Sum([]byte(utf16le))
	md5hash := hex.EncodeToString(hash[:])

	return challenge + "-" + md5hash, nil
}
