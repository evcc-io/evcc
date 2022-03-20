package charger

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/jpfielding/go-http-digest/pkg/digest"
	"github.com/mergermarket/go-pkcs7"
)

// Chiper stores the Tapo handshake response cipher
type TapoCipher struct {
	key []byte
	val []byte
}

// TP-Link Tapo charger implementation
type Tapo struct {
	*request.Helper
	log          *util.Logger
	uri          string
	email        string
	password     string
	cipher       *TapoCipher
	sessionID    string
	token        *string
	standbypower float64
}

func init() {
	registry.Add("tapo", NewTapoFromConfig)
}

// NewTapoFromConfig creates a Tapo charger from generic config
func NewTapoFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI          string
		User         string
		Password     string
		StandbyPower float64
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	return NewTapo(cc.URI, cc.User, cc.Password, cc.StandbyPower)
}

// NewTapo creates Tapo charger
func NewTapo(uri, user, password string, standbypower float64) (*Tapo, error) {
	for _, suffix := range []string{"/", "/app"} {
		uri = strings.TrimSuffix(uri, suffix)
	}

	log := util.NewLogger("tapo")
	client := request.NewHelper(log)

	c := &Tapo{
		Helper:       client,
		log:          log,
		standbypower: standbypower,
	}

	c.Client.Transport = request.NewTripper(log, transport.Insecure())

	if user == "" || password == "" {
		return c, fmt.Errorf("missing user/password")
	}

	// TP-Link Tapo API
	// https://k4czp3r.xyz/reverse-engineering/tp-link/tapo/2020/10/15/reverse-engineering-tp-link-tapo.html
	c.uri = fmt.Sprintf("%s/app", util.DefaultScheme(uri, "http"))
	if user != "" {
		c.Client.Transport = digest.NewTransport(user, password, c.Client.Transport)
	}

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *Tapo) Enabled() (bool, error) {
	return true, nil
}

// Enable implements the api.Charger interface
func (c *Tapo) Enable(enable bool) error {
	return nil
}

// MaxCurrent implements the api.Charger interface
func (c *Tapo) MaxCurrent(current int64) error {
	return nil
}

// Status implements the api.Charger interface
func (c *Tapo) Status() (api.ChargeStatus, error) {
	res := api.StatusB
	return res, nil
}

var _ api.Meter = (*Tapo)(nil)

// CurrentPower implements the api.Meter interface
func (c *Tapo) CurrentPower() (float64, error) {
	return 0, nil
}

// TapoRSAKeyGen generates handshake RSA key/value pair
func TapoRSAKeyGen(rsabits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, rsabits)
	if err != nil {
		return nil, nil, err
	}

	return key, key.Public().(*rsa.PublicKey), nil
}

func TapoRSAPEMDump(pubKey *rsa.PublicKey) ([]byte, error) {
	pubKeyPKIX, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(
		&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubKeyPKIX,
		},
	), nil
}

func (c *Tapo) TapoHandshake() error {
	_, pubKey, err := TapoRSAKeyGen(1024)
	if err != nil {
		return err
	}

	pubPEM, err := TapoRSAPEMDump(pubKey)
	if err != nil {
		return err
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"method": "handshake",
		"params": map[string]interface{}{
			"key":             string(pubPEM),
			"requestTimeMils": 0,
		},
	})

	return fmt.Errorf(string(payload))

}

func (tc *TapoCipher) TapoEncrypt(payload []byte) []byte {
	block, _ := aes.NewCipher(tc.key)
	encrypter := cipher.NewCBCEncrypter(block, tc.val)

	paddedPayload, _ := pkcs7.Pad(payload, aes.BlockSize)
	encryptedPayload := make([]byte, len(paddedPayload))
	encrypter.CryptBlocks(encryptedPayload, paddedPayload)

	return encryptedPayload
}

func (tc *TapoCipher) TapoDecrypt(payload []byte) []byte {
	block, _ := aes.NewCipher(tc.key)
	encrypter := cipher.NewCBCDecrypter(block, tc.val)

	decryptedPayload := make([]byte, len(payload))
	encrypter.CryptBlocks(decryptedPayload, payload)

	unpaddedPayload, _ := pkcs7.Unpad(decryptedPayload, aes.BlockSize)

	return unpaddedPayload
}
