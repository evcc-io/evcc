package leapmotor

import (
	"encoding/base64"
	"testing"
)

func makeJWT(username string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"user_name":"` + username + `"}`))
	return header + "." + payload + ".fakesig"
}

func TestExtractSessionDeviceID(t *testing.T) {
	t.Run("extracts third field", func(t *testing.T) {
		got := extractSessionDeviceID(makeJWT("user@x.com,region,dev-001,extra"))
		if got == nil || *got != "dev-001" {
			t.Errorf("got %v, want dev-001", got)
		}
	})

	t.Run("returns nil when fewer than 3 fields", func(t *testing.T) {
		if extractSessionDeviceID(makeJWT("a,b")) != nil {
			t.Error("expected nil")
		}
	})

	t.Run("returns nil when third field is empty", func(t *testing.T) {
		if extractSessionDeviceID(makeJWT("a,b,,d")) != nil {
			t.Error("expected nil for empty deviceID field")
		}
	})

	t.Run("returns nil for invalid token", func(t *testing.T) {
		if extractSessionDeviceID("not.a.token") != nil {
			t.Error("expected nil for unparseable token")
		}
	})
}

func TestDeriveSignKey(t *testing.T) {
	key, err := deriveSignKey("ikm-value", "salt-value", "info-value")
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(key))
	}
	key2, _ := deriveSignKey("ikm-value", "salt-value", "info-value")
	if string(key) != string(key2) {
		t.Error("not deterministic")
	}
	key3, _ := deriveSignKey("other-ikm", "salt-value", "info-value")
	if string(key) == string(key3) {
		t.Error("different ikm should produce different key")
	}
}

func TestP12MemoryEncode(t *testing.T) {
	cases := [][]byte{
		[]byte("short"),
		[]byte("exactly16bytesxx"),
		make([]byte, 33),
	}
	for _, input := range cases {
		out := p12MemoryEncode(input)
		if len(out)%16 != 0 {
			t.Errorf("output not block-aligned for input len %d: got %d bytes", len(input), len(out))
		}
		if len(out) < len(input) {
			t.Errorf("output shorter than input")
		}
	}
}

func TestDeriveP12Password(t *testing.T) {
	pwd := deriveP12Password("accountID123", "uid456")
	if pwd == "" {
		t.Error("expected non-empty password")
	}
	if len(pwd) > 15 {
		t.Errorf("password too long: %d", len(pwd))
	}
	if deriveP12Password("accountID123", "uid456") != pwd {
		t.Error("not deterministic")
	}
	if deriveP12Password("other", "uid456") == pwd {
		t.Error("different accountID should produce different password")
	}
}
