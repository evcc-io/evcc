package requests

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func Decorate(req *http.Request) error {
	req.Header.Set("tenant-id", TENANT_ID)
	req.Header.Set("user-type", USER_TYPE)
	req.Header.Set("app-content-encrypted", CONTENT_ENCRYPTED)
	req.Header.Set("Authorization", PARAM_AUTHENTICATION)

	return nil
}

func CreateRequest(
	endpoint string,
	httpMethod string,
	request string,
	contentType string,
	token string,
	eventId string,
) (*http.Request, error) {
	appSendDate := time.Now().UnixMilli()

	if len(request) != 0 {
		request = EncryptRequest(
			endpoint,
			appSendDate,
			TENANT_ID,
			token,
			request,
			contentType)
	}

	bodyReader := bytes.NewReader([]byte(request))
	req, err := http.NewRequest(httpMethod, endpoint, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("app-send-date", strconv.FormatInt(appSendDate, 10))
	req.Header.Set("original-content-type", contentType)

	if len(token) != 0 {
		req.Header.Set("blade-auth", token)
	}
	if len(eventId) != 0 {
		req.Header.Set("event-id", eventId)
	}

	replace := strings.Replace(endpoint, BASE_URL_P, "/", -1)

	req.Header.Set("app-verification-string",
		CalculateRequestVerification(
			replace,
			appSendDate,
			TENANT_ID,
			contentType,
			request,
			token))

	return req, nil
}

func DecryptAnswer(resp *http.Response) ([]byte, error) {
	var body string
	var bodyRaw []byte
	var err error
	if resp.StatusCode == http.StatusOK {
		bodyRaw, err = io.ReadAll(resp.Body)
		if err != nil {
			return bodyRaw, err
		}
		if resp.Header.Get("app-content-encrypted") == "1" {
			body = DecryptResponse(
				resp.Header.Get("app-send-date"),
				resp.Header.Get("original-content-type"),
				string(bodyRaw))
		} else {
			body = string(bodyRaw)
		}
	} else {
		err = fmt.Errorf(resp.Status)
	}

	return []byte(body), err
}
