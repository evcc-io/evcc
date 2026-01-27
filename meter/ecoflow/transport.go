package ecoflow

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util/transport"
	"github.com/samber/lo"
)

// NewAuthTransport creates HMAC-SHA256 signing transport
func NewAuthTransport(base http.RoundTripper, accessKey, secretKey string) http.RoundTripper {
	if base == nil {
		base = transport.Default()
	}

	return &transport.Decorator{
		Base: base,
		Decorator: func(req *http.Request) error {
			nonce := lo.RandomString(6, lo.NumbersCharset)
			timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())

			// Build signature from sorted query params
			keys := lo.Keys(req.URL.Query())
			slices.Sort(keys)

			parts := lo.Map(keys, func(k string, _ int) string {
				return k + "=" + req.URL.Query().Get(k)
			})

			signStr := strings.Join(parts, "&")
			if signStr != "" {
				signStr += "&"
			}
			signStr += fmt.Sprintf("accessKey=%s&nonce=%s&timestamp=%s", accessKey, nonce, timestamp)

			signature := hmacSHA256(signStr, secretKey)

			req.Header.Set("accessKey", accessKey)
			req.Header.Set("nonce", nonce)
			req.Header.Set("timestamp", timestamp)
			req.Header.Set("sign", signature)

			return nil
		},
	}
}

func hmacSHA256(data, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
