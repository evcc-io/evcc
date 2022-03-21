package charger

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/tapo"
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
	client       *http.Client
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

func TapoRSAPEMDump(publicKey *rsa.PublicKey) ([]byte, error) {
	publicKeyPKIX, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(
		&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKeyPKIX,
		},
	), nil
}

func TapoErrorCode(errorCode int) error {
	if errorCode != 0 {
		return errors.New(fmt.Sprintf("Got error code %d", errorCode))
	}

	return nil
}

func (c *Tapo) TapoHandshake() error {
	privateKey, publicKey, err := TapoRSAKeyGen(1024)
	if err != nil {
		return err
	}

	pubPEM, err := TapoRSAPEMDump(publicKey)
	if err != nil {
		return err
	}

	data, _ := json.Marshal(map[string]interface{}{
		"method": "handshake",
		"params": map[string]interface{}{
			"key":             string(pubPEM),
			"requestTimeMils": 0,
		},
	})

	var deviceResp tapo.DeviceResponse

	resp, err := http.Post(c.uri, "", bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(&deviceResp)
	if err = TapoErrorCode(deviceResp.ErrorCode); err != nil {
		return err
	}

	encryptedEncryptionKey, _ := base64.StdEncoding.DecodeString(deviceResp.Result.Key)
	encryptionKey, _ := rsa.DecryptPKCS1v15(rand.Reader, privateKey, []byte(encryptedEncryptionKey))

	c.cipher = &TapoCipher{
		key: encryptionKey[:16],
		val: encryptionKey[16:],
	}

	c.sessionID = strings.Split(resp.Header.Get("Set-Cookie"), ";")[0]

	return fmt.Errorf("data:\n%v\nkey:\n%s\nval:\n%s\nsessionId:\n%s\n", string(data), string(c.cipher.key), string(c.cipher.val), c.sessionID)
}

func (c *Tapo) TapoLogin() error {
	err := c.TapoHandshake()
	if err != nil {
		return err
	}

	h := sha1.New()
	h.Write([]byte(c.email))

	data, _ := json.Marshal(map[string]interface{}{
		"method": "login_device",
		"params": map[string]interface{}{
			"username": base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(h.Sum(nil)))),
			"password": base64.StdEncoding.EncodeToString([]byte(c.password)),
		},
	})

	var deviceResp tapo.DeviceResponse

	resp, err := http.Post(c.uri, "", bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(&deviceResp)

	return nil
}

func (c *Tapo) TapoRequest(data []byte) ([]byte, error) {
	securedData, _ := json.Marshal(map[string]interface{}{
		"method": "securePassthrough",
		"params": map[string]interface{}{
			"request": base64.StdEncoding.EncodeToString(c.cipher.TapoEncrypt(data)),
		},
	})

	req, _ := http.NewRequest("POST", c.uri, bytes.NewBuffer(securedData))
	req.Header.Set("Cookie", c.sessionID)
	req.Close = true

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var deviceResp tapo.DeviceResponse

	json.NewDecoder(resp.Body).Decode(&deviceResp)

	if err = TapoErrorCode(deviceResp.ErrorCode); err != nil {
		return nil, err
	}

	encryptedResponse, _ := base64.StdEncoding.DecodeString(deviceResp.Result.Response)

	return c.cipher.TapoDecrypt(encryptedResponse), nil
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
