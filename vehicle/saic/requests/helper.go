package requests

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"hash"
)

func sum(hash hash.Hash, value string) string {
	if len(value) == 0 {
		return ""
	}
	hash.Write([]byte(value))
	return hex.EncodeToString(hash.Sum(nil))
}

func Md5(value string) string {
	return sum(md5.New(), value)
}

func Sha1(value string) string {
	return sum(sha1.New(), value)
}

func Sha256(value string) string {
	return sum(sha256.New(), value)
}

func HmacSha256(secret string, message string) string {
	if len(secret) == 0 || len(message) == 0 {
		return ""
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}
