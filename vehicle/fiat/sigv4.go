package fiat

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

// signV4 adds the AWS Signature Version 4 authorization to req. Like the aws sdk it signs
// host, content-length and every header already set, except the hop-by-hop ones.
func signV4(req *http.Request, accessKey, secret, token, region, service, payloadHash string, now time.Time) {
	amzDate := now.UTC().Format("20060102T150405Z")
	date := amzDate[:8]

	req.Header.Set("X-Amz-Date", amzDate)
	if token != "" {
		req.Header.Set("X-Amz-Security-Token", token)
	}

	host := req.URL.Host
	if req.Host != "" {
		host = req.Host
	}

	headers := map[string]string{"host": host}
	if req.ContentLength > 0 {
		headers["content-length"] = strconv.FormatInt(req.ContentLength, 10)
	}
	for k, v := range req.Header {
		switch k {
		case "Authorization", "User-Agent", "X-Amzn-Trace-Id", "Expect", "Transfer-Encoding", "Content-Length":
			continue
		}
		vals := make([]string, len(v))
		for i, s := range v {
			vals[i] = strings.Join(strings.Fields(s), " ")
		}
		headers[strings.ToLower(k)] = strings.Join(vals, ",")
	}

	names := slices.Sorted(maps.Keys(headers))

	var canonicalHeaders strings.Builder
	for _, k := range names {
		canonicalHeaders.WriteString(k + ":" + headers[k] + "\n")
	}
	signedHeaders := strings.Join(names, ";")

	query := req.URL.Query()
	for k := range query {
		slices.Sort(query[k])
	}

	// paths here are plain, the double encoding of reserved characters is not needed
	path := req.URL.EscapedPath()
	if path == "" {
		path = "/"
	}

	canonical := strings.Join([]string{
		req.Method,
		path,
		strings.ReplaceAll(query.Encode(), "+", "%20"),
		canonicalHeaders.String(),
		signedHeaders,
		payloadHash,
	}, "\n")

	scope := strings.Join([]string{date, region, service, "aws4_request"}, "/")
	sum := sha256.Sum256([]byte(canonical))
	toSign := strings.Join([]string{"AWS4-HMAC-SHA256", amzDate, scope, hex.EncodeToString(sum[:])}, "\n")

	key := []byte("AWS4" + secret)
	for _, s := range []string{date, region, service, "aws4_request"} {
		key = hmacSHA256(key, s)
	}

	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+accessKey+"/"+scope+
		", SignedHeaders="+signedHeaders+", Signature="+hex.EncodeToString(hmacSHA256(key, toSign)))
}

func hmacSHA256(key []byte, s string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(s))
	return h.Sum(nil)
}
