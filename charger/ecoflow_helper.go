package charger

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
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
	// Generate random number in range [100000, 999999]
	// Using big.NewInt(900000) ensures max value is 900000-1, then add 100000
	max := big.NewInt(900000)
	randomBig, err := rand.Int(rand.Reader, max)
	if err != nil {
		// Fallback: use nanosecond-based nonce
		return strconv.FormatInt((time.Now().UnixNano()%900000)+100000, 10)
	}
	return strconv.FormatInt(randomBig.Int64()+100000, 10)
}

// ecoflowHmacSHA256 generates HMAC-SHA256 signature from the full signature string
func ecoflowHmacSHA256(signatureString, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(signatureString))
	return hex.EncodeToString(h.Sum(nil))
}

// NewEcoFlowAuthTransport creates authenticated HTTP transport with HMAC-SHA256 signing
func NewEcoFlowAuthTransport(accessKey, secretKey string) http.RoundTripper {
	return &ecoflowAuthTransport{
		base:      transport.Default(),
		accessKey: accessKey,
		secretKey: secretKey,
	}
}
