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
	log             *util.Logger
	URI             string
	EncodedUser     string
	EncodedPassword string
	Cipher          *ConnectionCipher
	SessionID       string
	Token           string
	TerminalUUID    string
}

// NewConnection creates a new Tapo device connection.
// User is encoded by using MessageDigest of SHA1 which is afterwards B64 encoded.
// Password is directly B64 encoded.
func NewConnection(uri, user, password string) *Connection {
	log := util.NewLogger("tapo")

	//lint:ignore
	h := sha1.New()
	_, _ = h.Write([]byte(user))
	userhash := hex.EncodeToString(h.Sum(nil))

	conn := &Connection{
		log:             log,
		Helper:          request.NewHelper(log),
		URI:             fmt.Sprintf("%s/app", util.DefaultScheme(uri, "http")),
		EncodedUser:     base64.StdEncoding.EncodeToString([]byte(userhash)),
		EncodedPassword: base64.StdEncoding.EncodeToString([]byte(password)),
	}

	conn.Client.Timeout = Timeout

	return conn
}

// Login provides the Tapo device session token and MAC address (TerminalUUID).
func (d *Connection) Login() error {
	err := d.Handshake()
	if err != nil {
		return err
	}

	req := map[string]interface{}{
		"method": "login_device",
		"params": map[string]interface{}{
			"username": d.EncodedUser,
			"password": d.EncodedPassword,
		},
	}

	res, err := d.DoSecureRequest(d.URI, req)
	if err != nil {
		return err
	}

	d.Token = res.Result.Token

	deviceResponse, err := d.ExecMethod("get_device_info", false)
	if err != nil {
		return err
	}

	d.TerminalUUID = deviceResponse.Result.MAC

	return nil
}

// Handshake provides the Tapo device session cookie and encryption cipher.
func (d *Connection) Handshake() error {
	privKey, pubKey, err := GenerateRSAKeys()
	if err != nil {
		return err
	}

	pubPEM, err := DumpRSAPEM(pubKey)
	if err != nil {
		return err
	}

	req, err := json.Marshal(map[string]interface{}{
		"method": "handshake",
		"params": map[string]interface{}{
			"key":             string(pubPEM),
			"requestTimeMils": 0,
		},
	})
	if err != nil {
		return err
	}

	resp, err := http.Post(d.URI, "application/json", bytes.NewBuffer(req))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var res DeviceResponse
	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}

	if err = d.CheckErrorCode(res.ErrorCode); err != nil {
		return err
	}

	encryptedEncryptionKey, err := base64.StdEncoding.DecodeString(res.Result.Key)
	if err != nil {
		return err
	}

	encryptionKey, err := rsa.DecryptPKCS1v15(rand.Reader, privKey, encryptedEncryptionKey)
	if err != nil {
		return err
	}

	d.Cipher = &ConnectionCipher{
		Key: encryptionKey[:16],
		Iv:  encryptionKey[16:],
	}

	cookie := strings.Split(resp.Header.Get("Set-Cookie"), ";")
	if len(cookie) == 0 {
		return errors.New("missing session cookie")
	}

	d.SessionID = cookie[0]

	return nil
}

// ExecMethod executes a Tapo device command method and provides the corresponding response.
func (d *Connection) ExecMethod(method string, deviceOn bool) (*DeviceResponse, error) {
	var req map[string]interface{}

	switch method {
	case "set_device_info":
		req = map[string]interface{}{
			"method": method,
			"params": map[string]interface{}{
				"device_on": deviceOn,
			},
			"requestTimeMils": int(time.Now().Unix() * 1000),
			"terminalUUID":    d.TerminalUUID,
		}
	default:
		req = map[string]interface{}{
			"method":          method,
			"requestTimeMils": int(time.Now().Unix() * 1000),
		}
	}

	res, err := d.DoSecureRequest(fmt.Sprintf("%s?token=%s", d.URI, d.Token), req)
	if err != nil {
		return nil, err
	}

	if method == "get_device_info" {
		res.Result.Nickname, err = base64Decode(res.Result.Nickname)
		if err != nil {
			return nil, err
		}

		res.Result.SSID, err = base64Decode(res.Result.SSID)
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

// DoSecureRequest executes a Tapo device request by encding the request and decoding its response.
func (d *Connection) DoSecureRequest(uri string, taporequest map[string]interface{}) (*DeviceResponse, error) {
	treq, err := json.Marshal(taporequest)
	if err != nil {
		return nil, err
	}

	d.log.TRACE.Printf("request: %s\n", string(treq))

	encryptedRequest, err := d.Cipher.Encrypt(treq)
	if err != nil {
		return nil, err
	}

	securedReq := map[string]interface{}{
		"method": "securePassthrough",
		"params": map[string]interface{}{
			"request": base64.StdEncoding.EncodeToString(encryptedRequest),
		},
	}

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(securedReq), map[string]string{
		"Cookie": d.SessionID,
	})
	if err != nil {
		return nil, err
	}

	var res *DeviceResponse
	if err = d.DoJSON(req, &res); err != nil {
		return nil, err
	}

	if err = d.CheckErrorCode(res.ErrorCode); err != nil {
		return nil, err
	}

	b64decodedResp, err := base64.StdEncoding.DecodeString(res.Result.Response)
	if err != nil {
		return nil, err
	}

	decryptedResponse, err := d.Cipher.Decrypt(b64decodedResp)
	if err != nil {
		return nil, err
	}

	d.log.TRACE.Printf("decrypted result: %v\n", string(decryptedResponse))

	var deviceResp *DeviceResponse
	if err = json.Unmarshal(decryptedResponse, &deviceResp); err != nil {
		return deviceResp, err
	}

	return deviceResp, nil
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

func (c *ConnectionCipher) Encrypt(payload []byte) ([]byte, error) {
	paddedPayload, err := pkcs7.Pad(payload, aes.BlockSize)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(c.Key)
	if err != nil {
		return nil, err
	}
	encrypter := cipher.NewCBCEncrypter(block, c.Iv)
	encryptedPayload := make([]byte, len(paddedPayload))
	encrypter.CryptBlocks(encryptedPayload, paddedPayload)

	return encryptedPayload, nil
}

func (c *ConnectionCipher) Decrypt(payload []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.Key)
	if err != nil {
		return nil, err
	}

	encrypter := cipher.NewCBCDecrypter(block, c.Iv)
	decryptedPayload := make([]byte, len(payload))

	encrypter.CryptBlocks(decryptedPayload, payload)

	unpaddedPayload, err := pkcs7.Unpad(decryptedPayload, aes.BlockSize)
	if err != nil {
		return nil, err
	}

	return unpaddedPayload, nil
}

func DumpRSAPEM(pubKey *rsa.PublicKey) ([]byte, error) {
	pubKeyPKIX, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, err
	}

	pubPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubKeyPKIX,
		},
	)

	return pubPEM, nil
}

func GenerateRSAKeys() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, nil, err
	}

	return key, key.Public().(*rsa.PublicKey), nil
}

func base64Decode(base64String string) (string, error) {
	decodedString, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return "", err
	}

	return string(decodedString), nil
}
