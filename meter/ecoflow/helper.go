package ecoflow

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/util/transport"
)

// ecoflowAuthTransport implements http.RoundTripper with HMAC-SHA256 signature
type ecoflowAuthTransport struct {
	base      http.RoundTripper
	accessKey string
	secretKey string
}

// RoundTrip implements http.RoundTripper interface
func (t *ecoflowAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	nonce := ecoflowGenerateNonce()
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())

	// Build signature string from query parameters
	var signStr string
	if req.URL.RawQuery != "" {
		signStr = req.URL.RawQuery
	}

	if signStr != "" {
		signStr += "&"
	}
	signStr += fmt.Sprintf("accessKey=%s&nonce=%s&timestamp=%s", t.accessKey, nonce, timestamp)

	// Create signature
	signature := ecoflowHmacSHA256(signStr, t.secretKey)

	// Set authorization headers (these go into the request, not URL params)
	req.Header.Set("accessKey", t.accessKey)
	req.Header.Set("nonce", nonce)
	req.Header.Set("timestamp", timestamp)
	req.Header.Set("sign", signature)

	return t.base.RoundTrip(req)
}

// ecoflowGenerateNonce generates a random 6-digit nonce (100000-999999)
func ecoflowGenerateNonce() string {
	// Generate 3 random bytes and convert to 6-digit number
	buf := make([]byte, 3)
	rand.Read(buf) // Never returns error per crypto/rand docs
	
	// Convert 3 bytes to value in range [0, 16777215]
	// Map to [100000, 999999] range (900000 possible values)
	val := uint32(buf[0])<<16 | uint32(buf[1])<<8 | uint32(buf[2])
	nonce := (val % 900000) + 100000
	return strconv.FormatUint(uint64(nonce), 10)
}

// ecoflowHmacSHA256 generates HMAC-SHA256 signature from the full signature string
func ecoflowHmacSHA256(signatureString, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(signatureString))
	return hex.EncodeToString(h.Sum(nil))
}

// NewEcoFlowAuthTransport creates authenticated HTTP transport with HMAC-SHA256 signing,
// wrapping the provided base RoundTripper to preserve existing HTTP settings (proxies, TLS, etc.).
func NewEcoFlowAuthTransport(base http.RoundTripper, accessKey, secretKey string) http.RoundTripper {
	// fall back to the default transport if no base has been configured
	if base == nil {
		base = transport.Default()
	}

	return &ecoflowAuthTransport{
		base:      base,
		accessKey: accessKey,
		secretKey: secretKey,
	}
}
