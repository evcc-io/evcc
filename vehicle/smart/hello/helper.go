package hello

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"time"

	"github.com/samber/lo"
)

func createSignature(method, path string, params url.Values, body io.Reader) (string, string, string, error) {
	nonce := lo.RandomString(16, lo.AlphanumericCharset)
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)

	md5Hash := "1B2M2Y8AsgTpgAmY7PhCfg=="
	if body != nil {
		bytes, err := io.ReadAll(body)
		if err != nil {
			return "", "", "", err
		}

		hash := md5.New()
		hash.Write(bytes)
		md5Hash = base64.StdEncoding.EncodeToString(hash.Sum(nil))
	}

	payload := fmt.Sprintf(`application/json;responseformat=3
x-api-signature-nonce:%s
x-api-signature-version:1.0

%s
%s
%s
%s
%s`, nonce, params.Encode(), md5Hash, ts, method, path)

	secret, err := base64.StdEncoding.DecodeString("NzRlNzQ2OWFmZjUwNDJiYmJlZDdiYmIxYjM2YzE1ZTk=")
	if err != nil {
		return "", "", "", err
	}

	mac := hmac.New(sha1.New, secret)
	mac.Write([]byte(payload))
	sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return nonce, ts, sign, nil
}

func responseError(err error, code ResponseCode, msg string, errS Error) error {
	var body error

	if code != 0 && code != ResponseOK {
		body = fmt.Errorf("%d: %s", code, msg)
	} else if errS.Code != 0 && errS.Code != ResponseOK {
		body = fmt.Errorf("%d: %s", errS.Code, errS.Message)
	}

	if err == nil {
		return body
	}

	return fmt.Errorf("%w: %s", err, body)
}
