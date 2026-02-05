package requests

import (
	"bytes"
	"errors"
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

func encryptRequest(resourcePath string, sendDate int64, tenant, token, body, contentType string) string {
	if len(body) == 0 {
		return ""
	}

	dateString := strconv.FormatInt(sendDate, 10)

	// tenant
	if len(resourcePath) != 0 {
		resourcePath = "/" + resourcePath
	}

	sb3 := Md5(resourcePath+tenant+token+USER_TYPE) + dateString + CONTENT_ENCRYPTED + contentType
	a2 := Md5(sb3)
	a3 := Md5(dateString)

	return Encrypt(body, a2, a3)
}

func calculateRequestVerification(
	resourcePath string,
	sendDate int64,
	tenant, contentType, bodyEncrypted, token string,
) string {
	dateString := strconv.FormatInt(sendDate, 10)
	str9 := resourcePath + tenant + token + USER_TYPE
	a2 := Md5(str9)
	str10 := dateString + CONTENT_ENCRYPTED + contentType
	a3 := Md5(a2 + str10)
	str11 := resourcePath + tenant + token + USER_TYPE + dateString + CONTENT_ENCRYPTED + contentType + bodyEncrypted

	a5 := Md5(a3 + dateString)

	return HmacSha256(a5, str11)
}

func CreateRequest(baseUrl, path, httpMethod, request, contentType, token, eventId string) (*http.Request, error) {
	sendDate := time.Now().UnixMilli()

	endpoint := baseUrl + path

	if len(request) != 0 {
		request = encryptRequest(
			path,
			sendDate,
			TENANT_ID,
			token,
			request,
			contentType)
	}

	req, err := http.NewRequest(httpMethod, endpoint, bytes.NewReader([]byte(request)))
	if err != nil {
		return nil, err
	}

	Decorate(req)
	req.Header.Set("app-send-date", strconv.FormatInt(sendDate, 10))
	req.Header.Set("original-content-type", contentType)

	if len(token) != 0 {
		req.Header.Set("blade-auth", token)
	}
	if len(eventId) != 0 {
		req.Header.Set("event-id", eventId)
	}

	replace := strings.Replace(endpoint, baseUrl, "/", -1)

	req.Header.Set("app-verification-string",
		calculateRequestVerification(
			replace,
			sendDate,
			TENANT_ID,
			contentType,
			request,
			token))

	return req, nil
}

func decryptResponse(timeStamp, contentType, cipherText string) string {
	if len(cipherText) == 0 {
		return ""
	}

	str4 := timeStamp + CONTENT_ENCRYPTED + contentType
	a2 := Md5(str4)
	hashedTimeStamp := Md5(timeStamp)

	return Decrypt(cipherText, a2, hashedTimeStamp)
}

func DecodeResponse(resp *http.Response) ([]byte, error) {
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return body, err
	}

	if resp.Header.Get("app-content-encrypted") != "1" {
		return body, nil
	}

	decoded := decryptResponse(
		resp.Header.Get("app-send-date"),
		resp.Header.Get("original-content-type"),
		string(body))

	return []byte(decoded), nil
}
