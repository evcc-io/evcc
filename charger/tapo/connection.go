package tapo

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
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/mergermarket/go-pkcs7"
)

// Tapo homepage + api reverse engineering results
// https://www.tapo.com/de/
// Credits to & inspired by:
// https://k4czp3r.xyz/reverse-engineering/tp-link/tapo/2020/10/15/reverse-engineering-tp-link-tapo.html
// https://github.com/fishbigger/TapoP100
// https://github.com/artemvang/p100-go

const Timeout = time.Second * 15

// Connection is the Tapo connection
type Connection struct {
	*request.Helper
	log *util.Logger
	*Settings
	EncodedUser     string
	EncodedPassword string
	Cipher          *ConnectionCipher
	SessionID       string
	Token           *string
	Client          *http.Client
	TerminalUUID    string
}

// NewConnection creates a new Tapo device connection.
// User is encoded by using MessageDigest of SHA1 which is afterwards B64 encoded.
// Password is directly B64 encoded.
func NewConnection(uri, user, password string) *Connection {
	log := util.NewLogger("tapo")

	settings := &Settings{
		URI:      fmt.Sprintf("%s/app", util.DefaultScheme(uri, "http")),
		User:     user,
		Password: password,
	}

	//lint:ignore
	h := sha1.New()
	_, _ = h.Write([]byte(user))
	userhash := hex.EncodeToString(h.Sum(nil))

	tapo := &Connection{
		log:             log,
		Helper:          request.NewHelper(log),
		Settings:        settings,
		EncodedUser:     base64.StdEncoding.EncodeToString([]byte(userhash)),
		EncodedPassword: base64.StdEncoding.EncodeToString([]byte(password)),
		Client:          &http.Client{Timeout: Timeout},
	}

	return tapo
}

// Login provides the Tapo device session token and MAC address (TerminalUUID).
func (d *Connection) Login() error {
	err := d.Handshake()
	if err != nil {
		return err
	}

	req, _ := json.Marshal(map[string]interface{}{
		"method": "login_device",
		"params": map[string]interface{}{
			"username": d.EncodedUser,
			"password": d.EncodedPassword,
		},
	})

	res, err := d.DoSecureRequest(d.URI, req)
	if err != nil {
		return err
	}

	var jsonResp DeviceResponse
	json.NewDecoder(bytes.NewBuffer(res)).Decode(&jsonResp)
	if err = d.CheckErrorCode(jsonResp.ErrorCode); err != nil {
		return err
	}

	d.Token = &jsonResp.Result.Token

	deviceResponse, err := d.ExecMethod("get_device_info", false)
	if err != nil {
		return err
	}

	d.TerminalUUID = deviceResponse.Result.MAC

	return nil
}

// Handshake provides the Tapo device session cookie and encryption cipher.
func (d *Connection) Handshake() error {
	privKey, pubKey := GenerateRSAKeys()

	pubPEM := DumpRSAPEM(pubKey)
	req, _ := json.Marshal(map[string]interface{}{
		"method": "handshake",
		"params": map[string]interface{}{
			"key":             string(pubPEM),
			"requestTimeMils": 0,
		},
	})

	res, err := http.Post(d.URI, "application/json", bytes.NewBuffer(req))
	if err != nil {
		return err
	}

	defer res.Body.Close()

	var jsonResp DeviceResponse
	json.NewDecoder(res.Body).Decode(&jsonResp)
	if err = d.CheckErrorCode(jsonResp.ErrorCode); err != nil {
		return err
	}

	encryptedEncryptionKey, _ := base64.StdEncoding.DecodeString(jsonResp.Result.Key)
	encryptionKey, _ := rsa.DecryptPKCS1v15(rand.Reader, privKey, encryptedEncryptionKey)
	d.Cipher = &ConnectionCipher{
		Key: encryptionKey[:16],
		Iv:  encryptionKey[16:],
	}

	d.SessionID = strings.Split(res.Header.Get("Set-Cookie"), ";")[0]

	return nil
}

// ExecMethod executes a Tapo device command method and provides the corresponding response.
func (d *Connection) ExecMethod(method string, deviceOn bool) (*DeviceResponse, error) {
	var req []byte
	switch method {
	case "set_device_info":
		req, _ = json.Marshal(map[string]interface{}{
			"method": method,
			"params": map[string]interface{}{
				"device_on": deviceOn,
			},
			"requestTimeMils": int(time.Now().Unix() * 1000),
			"terminalUUID":    d.TerminalUUID,
		})
	default:
		req, _ = json.Marshal(map[string]interface{}{
			"method":          method,
			"requestTimeMils": int(time.Now().Unix() * 1000),
		})
	}

	res, err := d.DoSecureRequest(fmt.Sprintf("%s?token=%s", d.URI, *d.Token), req)
	if err != nil {
		return nil, err
	}

	d.log.TRACE.Printf("%v", string(res))

	tapoResp := &DeviceResponse{}
	json.NewDecoder(bytes.NewBuffer(res)).Decode(tapoResp)

	if method == "get_device_info" {
		tapoResp.Result.Nickname = base64Decode(tapoResp.Result.Nickname)
		tapoResp.Result.SSID = base64Decode(tapoResp.Result.SSID)
	}

	return tapoResp, nil
}

// DoSecureRequest executes a Tapo device request by encding the request and decoding its response.
func (d *Connection) DoSecureRequest(uri string, taporequest []byte) ([]byte, error) {
	securedReq := map[string]interface{}{
		"method": "securePassthrough",
		"params": map[string]interface{}{
			"request": base64.StdEncoding.EncodeToString(d.Cipher.Encrypt(taporequest)),
		},
	}

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(securedReq), map[string]string{
		"Cookie": d.SessionID,
	})

	var res DeviceResponse
	if err == nil {
		err = d.DoJSON(req, &res)
	}

	if err = d.CheckErrorCode(res.ErrorCode); err != nil {
		return []byte(fmt.Sprintf("{\"error_code\":%v}", res.ErrorCode)), err
	}

	encryptedResponse, _ := base64.StdEncoding.DecodeString(res.Result.Response)

	return d.Cipher.Decrypt(encryptedResponse), nil
}

// Tapo helper functions

func (d *Connection) CheckErrorCode(errorCode int) error {
	errorDesc := map[int]string{
		0:     "Success",
		-1002: "Incorrect Request/Method",
		-1003: "JSON formatting error ",
		-1010: "Invalid Public Key Length",
		-1012: "Invalid terminalUUID",
		-1501: "Invalid Request or Credentials",
	}

	if errorCode != 0 {
		return errors.New(fmt.Sprintf("Tapo error %d: %s", errorCode, errorDesc[errorCode]))
	}

	return nil
}

func (c *ConnectionCipher) Encrypt(payload []byte) []byte {
	block, _ := aes.NewCipher(c.Key)
	encrypter := cipher.NewCBCEncrypter(block, c.Iv)

	paddedPayload, _ := pkcs7.Pad(payload, aes.BlockSize)
	encryptedPayload := make([]byte, len(paddedPayload))
	encrypter.CryptBlocks(encryptedPayload, paddedPayload)

	return encryptedPayload
}

func (c *ConnectionCipher) Decrypt(payload []byte) []byte {
	block, _ := aes.NewCipher(c.Key)
	encrypter := cipher.NewCBCDecrypter(block, c.Iv)

	decryptedPayload := make([]byte, len(payload))
	encrypter.CryptBlocks(decryptedPayload, payload)

	unpaddedPayload, _ := pkcs7.Unpad(decryptedPayload, aes.BlockSize)

	return unpaddedPayload
}

func DumpRSAPEM(pubKey *rsa.PublicKey) (pubPEM []byte) {
	pubKeyPKIX, _ := x509.MarshalPKIXPublicKey(pubKey)

	pubPEM = pem.EncodeToMemory(
		&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubKeyPKIX,
		},
	)

	return
}

func GenerateRSAKeys() (*rsa.PrivateKey, *rsa.PublicKey) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}

	return key, key.Public().(*rsa.PublicKey)
}

func base64Decode(base64String string) string {
	decodedString, _ := base64.StdEncoding.DecodeString(base64String)
	return string(decodedString)
}
