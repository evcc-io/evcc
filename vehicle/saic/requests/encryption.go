package requests

import (
	"strconv"
	"strings"
)

func CalculateRequestVerification(
	resourcePath string,
	sendDate int64,
	tenant,
	contentType,
	bodyEncrypted,
	token string,
) string {
	dateString := strconv.FormatInt(sendDate, 10)
	str9 := resourcePath + tenant + token + USER_TYPE
	a2 := Md5(str9)
	str10 := dateString + CONTENT_ENCRYPTED + contentType
	a3 := Md5(a2 + str10)
	str11 := resourcePath + tenant + token + USER_TYPE + dateString + CONTENT_ENCRYPTED + contentType + bodyEncrypted

	a5 := Md5(a3 + dateString)

	if len(a5) != 0 && len(str11) != 0 {
		return HmacSha256(a5, str11)
	} else {
		return ""
	}
}

func DecryptResponse(timeStamp, contentType, cipherText string) string {
	str4 := timeStamp + CONTENT_ENCRYPTED + contentType
	a2 := Md5(str4)
	hashedTimeStamp := Md5(timeStamp)

	if len(cipherText) != 0 {
		return Decrypt(cipherText, a2, hashedTimeStamp)
	}
	return ""
}

func CalculateResponseVerification(str, str2, str3 string) string {
	str4 := str + CONTENT_ENCRYPTED + str2
	a2 := Md5(str4)
	str5 := str + CONTENT_ENCRYPTED + str2 + str3
	a4 := Md5(a2 + str)
	return HmacSha256(a4, str5)
}

func EncryptRequest(url string, time int64, tenant, token, body, contentType string) string {
	sendDate := strconv.FormatInt(time, 10)
	// tenant
	replace := ""
	if len(url) != 0 {
		replace = strings.Replace(url, BASE_URL_P, "/", -1)
	}

	encryptedBody := ""

	if len(body) != 0 {
		sb3 := Md5(replace+tenant+token+USER_TYPE) + sendDate + CONTENT_ENCRYPTED + contentType
		a2 := Md5(sb3)
		a3 := Md5(sendDate)
		if len(body) != 0 && len(a2) != 0 && len(a3) != 0 {
			encryptedBody = Encrypt(body, a2, a3)
		}
	}

	return encryptedBody
}

func DecryptRequest(url string, time int64, tenant, token, body, contentType string) string {
	timeStamp := strconv.FormatInt(time, 10)
	resourcePath := strings.Replace(url, BASE_URL_P, "/", -1)
	if len(body) != 0 {
		sb3 := Md5(resourcePath+tenant+token+USER_TYPE) + timeStamp + CONTENT_ENCRYPTED + contentType
		a2 := Md5(sb3)
		timeStampHash := Md5(timeStamp)
		return Decrypt(body, a2, timeStampHash)
	}
	return ""
}
