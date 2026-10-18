package fiat

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// expected values recorded from aws-sdk-go-v2 v1.47.1 SignHTTP for the same requests
func TestSignV4(t *testing.T) {
	const (
		accessKey = "ASIAEXAMPLE"
		secret    = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
		token     = "IQoJb3JpZ2luX2VjEXAMPLE"
		scope     = "Credential=ASIAEXAMPLE/20261015/eu-west-1/execute-api/aws4_request"
		get       = "SignedHeaders=accept;clientrequestid;content-type;host;locale;x-amz-date;x-amz-security-token;x-api-key;x-clientapp-version;x-originator-type"
		post      = "SignedHeaders=accept;clientrequestid;content-length;content-type;host;locale;x-amz-date;x-amz-security-token;x-api-key;x-clientapp-version;x-originator-type"
	)
	now := time.Date(2026, 10, 15, 13, 45, 0, 0, time.UTC)

	for _, tc := range []struct{ method, uri, body, headers, signature string }{
		{http.MethodGet, ApiURI + "/v4/accounts/0123456789abcdef/vehicles?stage=ALL", "",
			get, "f1bcde2296b3473715e6886a63c30da9bc0c7b4edc56b323fedfe339d0e5557f"},
		{http.MethodPost, ApiURI + "/v1/accounts/0123456789abcdef/vehicles/ZFA1234567890/remote", `{"command":"RON","pin":"c2VjcmV0"}`,
			post, "3f1e0e55fede45545998ac824b6eee14145c6e8222c74c11f7c46caa55ffce57"},
		{http.MethodPost, AuthURI + "/v1/accounts/0123456789abcdef/ignite/pin/authenticate", `{"pin":"c2VjcmV0"}`,
			post, "c0a44f05b8573f8415cf4838a42bc0b81703d22f402809cd790f2cd095afada7"},
	} {
		var body io.ReadSeeker
		if tc.body != "" {
			body = strings.NewReader(tc.body)
		}
		req, err := request.New(tc.method, tc.uri, body, map[string]string{
			"Content-Type":        "application/json",
			"Accept":              "application/json",
			"X-Clientapp-Version": "1.0",
			"ClientrequestId":     "abcdefghijklmnop",
			"X-Api-Key":           XApiKey,
			"X-Originator-Type":   "web",
			"locale":              "de_de",
		})
		require.NoError(t, err)

		hash, err := hashBody(body)
		require.NoError(t, err)
		signV4(req, accessKey, secret, token, Region, "execute-api", hash, now)

		assert.Equal(t, "20261015T134500Z", req.Header.Get("X-Amz-Date"), tc.uri)
		assert.Equal(t, token, req.Header.Get("X-Amz-Security-Token"), tc.uri)
		assert.Equal(t, "AWS4-HMAC-SHA256 "+scope+", "+tc.headers+", Signature="+tc.signature, req.Header.Get("Authorization"), tc.uri)
	}
}
