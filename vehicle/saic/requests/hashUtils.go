package requests

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
)

func Md5(value string) string {
	if len(value) == 0 {
		return ""
	}
	result := md5.Sum([]byte(value))
	return hex.EncodeToString(result[:])
}

func Sha1(value string) string {
	if len(value) == 0 {
		return ""
	}
	result := sha1.Sum([]byte(value))
	return hex.EncodeToString(result[:])
}

func Sha256(value string) string {
	if len(value) == 0 {
		return ""
	}
	result := sha256.Sum256([]byte(value))
	return hex.EncodeToString(result[:])
}
