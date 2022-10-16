package loginapps

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMarshalJSON(t *testing.T) {
	tm := time.Now()
	tok := &Token{
		TokenType:    "type",
		AccessToken:  "access",
		RefreshToken: "refresh",
		Expiry:       tm,
	}

	b, err := json.Marshal(tok)
	assert.Nil(t, err)
	assert.True(t, strings.HasPrefix(string(b), `{"type":"type","accesstoken":"access","refreshtoken":"refresh","expiry":`))
}

func TestUnmarshalJSON(t *testing.T) {
	str := `{"type":"","accesstoken":"access","refreshtoken":"refresh"}`

	var tok *Token
	if err := json.Unmarshal([]byte(str), &tok); err != nil {
		t.Error(err)
	}

	assert.Equal(t, "access", tok.AccessToken)
	assert.Equal(t, "refresh", tok.RefreshToken)
	assert.Equal(t, "bearer", tok.TokenType)
	assert.NotEmpty(t, tok.Expiry)
}
