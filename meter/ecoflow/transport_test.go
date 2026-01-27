package ecoflow

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthTransport(t *testing.T) {
	accessKey := "myAccessKey"
	secretKey := "mySecretKey"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers
		assert.Equal(t, accessKey, r.Header.Get("accessKey"))
		assert.NotEmpty(t, r.Header.Get("nonce"))
		assert.NotEmpty(t, r.Header.Get("timestamp"))
		assert.NotEmpty(t, r.Header.Get("sign"))

		// Check nonce format (6 digits)
		nonce := r.Header.Get("nonce")
		assert.Len(t, nonce, 6)

		// Check timestamp
		timestamp := r.Header.Get("timestamp")
		assert.NotEmpty(t, timestamp)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := NewAuthTransport(http.DefaultTransport, accessKey, secretKey)
	client := &http.Client{Transport: transport}

	req, err := http.NewRequest("GET", server.URL+"?param1=value1", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHmacSHA256(t *testing.T) {
	data := "testData"
	secret := "testSecret"
	// echo -n "testData" | openssl dgst -sha256 -hmac "testSecret"
	expected := "cf12e86d709b0a720cf2f129b914369fd14d979f5d44ec4eb04019189df57f01"
	
	result := hmacSHA256(data, secret)
	assert.Equal(t, expected, result)
}
