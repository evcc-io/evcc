package vaillant

import (
	"bytes"
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// low cost and one-byte keyPrefix so the proof-of-work solves almost instantly
func TestSolveAltcha(t *testing.T) {
	challenge := []byte(`{
		"parameters": {
			"algorithm": "PBKDF2/SHA-256",
			"cost": 10,
			"keyLength": 32,
			"keyPrefix": "00",
			"nonce": "19398d35354f4059a03226019c7b9915",
			"salt": "df78709ec7a451e5eacc099b09e2e9a7"
		},
		"signature": "some-server-signature"
	}`)

	payload, err := solveAltcha(challenge)
	require.NoError(t, err)

	res, err := base64.StdEncoding.DecodeString(payload)
	require.NoError(t, err)

	var decoded struct {
		Challenge struct {
			Parameters struct {
				Cost      int    `json:"cost"`
				KeyLength int    `json:"keyLength"`
				KeyPrefix string `json:"keyPrefix"`
				Nonce     string `json:"nonce"`
				Salt      string `json:"salt"`
			} `json:"parameters"`
			Signature string `json:"signature"`
		} `json:"challenge"`
		Solution struct {
			Counter    uint32 `json:"counter"`
			DerivedKey string `json:"derivedKey"`
		} `json:"solution"`
	}
	require.NoError(t, json.Unmarshal(res, &decoded))
	require.Equal(t, "some-server-signature", decoded.Challenge.Signature)

	p := decoded.Challenge.Parameters
	nonce, err := hex.DecodeString(p.Nonce)
	require.NoError(t, err)
	salt, err := hex.DecodeString(p.Salt)
	require.NoError(t, err)
	prefix, err := hex.DecodeString(p.KeyPrefix)
	require.NoError(t, err)

	password := binary.BigEndian.AppendUint32(nonce, decoded.Solution.Counter)
	key, err := pbkdf2.Key(sha256.New, string(password), salt, p.Cost, p.KeyLength)
	require.NoError(t, err)

	require.Equal(t, hex.EncodeToString(key), decoded.Solution.DerivedKey)
	require.True(t, bytes.HasPrefix(key, prefix))
}
