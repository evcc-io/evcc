package tesla

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/http"

	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/server/service"
	"github.com/teslamotors/vehicle-command/pkg/protocol"
)

// tesla fetches the command signing public key from this path
const publicKeyPath = "/.well-known/appspecific/com.tesla.3p.public-key.pem"

const privateKeySetting = "tesla.privateKey"

func init() {
	service.RegisterPublic(publicKeyPath, http.HandlerFunc(publicKeyHandler))
}

func loadPrivateKey() (*ecdsa.PrivateKey, error) {
	s, err := settings.String(privateKeySetting)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode([]byte(s))
	if block == nil {
		return nil, errors.New("invalid private key")
	}

	return x509.ParseECPrivateKey(block.Bytes)
}

func ensurePrivateKey() (*ecdsa.PrivateKey, error) {
	if key, err := loadPrivateKey(); err == nil {
		return key, nil
	}

	return generatePrivateKey()
}

func generatePrivateKey() (*ecdsa.PrivateKey, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}

	settings.SetString(privateKeySetting, string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})))

	// tesla holds the public key from now on, losing the private key means pairing again
	if db.Instance != nil {
		if err := settings.Persist(); err != nil {
			return nil, err
		}
	}

	return key, nil
}

// SigningKey returns the command signing key in vehicle-command format
func SigningKey() (protocol.ECDHPrivateKey, error) {
	key, err := loadPrivateKey()
	if err != nil {
		return nil, errors.New("missing signing key, connect the Tesla account first")
	}

	scalar, err := key.Bytes()
	if err != nil {
		return nil, err
	}

	skey := protocol.UnmarshalECDHPrivateKey(scalar)
	if skey == nil {
		return nil, errors.New("invalid signing key")
	}

	return skey, nil
}

func publicKeyPEM(key *ecdsa.PrivateKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), nil
}

func publicKeyHandler(w http.ResponseWriter, r *http.Request) {
	key, err := loadPrivateKey()
	if err != nil {
		http.NotFound(w, r)
		return
	}

	b, err := publicKeyPEM(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-pem-file")
	_, _ = w.Write(b)
}
