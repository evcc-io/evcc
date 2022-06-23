package carwings

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/crypto/blowfish"
)

const (
	initialAppStrings = "9s5rfKVuMrT03RtzajWNcA"
	BaseURL           = "https://gdcportalgw.its-mo.com/api_v200413_NE/gdc/"
)

type Identity struct {
	*request.Helper
	SessionID string
}

// NewIdentity creates Carwings identity
func NewIdentity(log *util.Logger) *Identity {
	return &Identity{
		Helper: request.NewHelper(log),
	}
}

func (v *Identity) Login(user, password string) (LoginResponse, error) {
	baseprm, err := v.getBaseprm()
	if err != nil {
		return LoginResponse{}, err
	}

	encpw, err := encrypt(password, baseprm)
	if err != nil {
		return LoginResponse{}, err
	}

	params := url.Values{}
	params.Set("initial_app_str", initialAppStrings)

	params.Set("UserId", user)
	params.Set("Password", encpw)
	params.Set("RegionCode", "NE")

	uri := fmt.Sprint(BaseURL + "UserLoginRequest.php")

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(params), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"User-Agent":   "",
	})
	if err != nil {
		return LoginResponse{}, err
	}

	var res LoginResponse
	err = v.DoJSON(req, &res)
	if err != nil {
		return LoginResponse{}, err
	}
	v.SessionID = res.VehicleInfo.CustomSessionID
	return res, nil
}

func (v *Identity) getBaseprm() (string, error) {
	uri := fmt.Sprint(BaseURL + "InitialApp_v2.php")
	params := url.Values{}
	params.Set("initial_app_str", initialAppStrings)
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(params), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"User-Agent":   "",
	})
	if err != nil {
		return "", err
	}
	var res Connect
	err = v.DoJSON(req, &res)
	if err != nil {
		return "", err
	}
	return res.Baseprm, nil
}

func pkcs5Padding(data []byte, blocksize int) []byte {
	padLen := blocksize - (len(data) % blocksize)
	padding := bytes.Repeat([]byte{byte(padLen)}, padLen)
	return append(data, padding...)
}

// Pads the source, does ECB Blowfish encryption on it, and returns a
// base64-encoded string.
func encrypt(s, key string) (string, error) {
	cipher, err := blowfish.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	src := []byte(s)
	src = pkcs5Padding(src, cipher.BlockSize())

	dst := make([]byte, len(src))
	pos := 0
	for pos < len(src) {
		cipher.Encrypt(dst[pos:], src[pos:])
		pos += cipher.BlockSize()
	}

	return base64.StdEncoding.EncodeToString(dst), nil
}
