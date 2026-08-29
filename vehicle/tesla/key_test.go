package tesla

import (
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/stretchr/testify/require"
)

func TestSigningKey(t *testing.T) {
	// no db in tests, blank the in-memory setting instead of deleting
	t.Cleanup(func() { settings.SetString(privateKeySetting, "") })

	rr := httptest.NewRecorder()
	publicKeyHandler(rr, httptest.NewRequest(http.MethodGet, publicKeyPath, nil))
	require.Equal(t, http.StatusNotFound, rr.Code)

	_, err := SigningKey()
	require.Error(t, err)

	key, err := ensurePrivateKey()
	require.NoError(t, err)

	again, err := ensurePrivateKey()
	require.NoError(t, err)
	require.True(t, key.Equal(again))

	skey, err := SigningKey()
	require.NoError(t, err)

	rr = httptest.NewRecorder()
	publicKeyHandler(rr, httptest.NewRequest(http.MethodGet, publicKeyPath, nil))
	require.Equal(t, http.StatusOK, rr.Code)

	block, _ := pem.Decode(rr.Body.Bytes())
	require.NotNil(t, block)
	require.Equal(t, "PUBLIC KEY", block.Type)

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	require.NoError(t, err)
	require.True(t, key.PublicKey.Equal(pub))

	ecdhPub, err := key.PublicKey.ECDH()
	require.NoError(t, err)
	require.Equal(t, ecdhPub.Bytes(), skey.PublicBytes())

	rotated, err := generatePrivateKey()
	require.NoError(t, err)
	require.False(t, key.Equal(rotated))

	again, err = ensurePrivateKey()
	require.NoError(t, err)
	require.True(t, rotated.Equal(again))
}
