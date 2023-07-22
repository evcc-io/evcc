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
	updated         time.Time
	lasttodayenergy int64
	energy          int64
}

// NewConnection creates a new Tapo device connection.
// User is encoded by using MessageDigest of SHA1 which is afterwards B64 encoded.
// Password is directly B64 encoded.
func NewConnection(uri, user, password string) (*Connection, error) {
	if uri == "" {
		return nil, errors.New("missing uri")
	}

	if user == "" || password == "" {
		return nil, fmt.Errorf("missing user or password")
	}

	for _, suffix := range []string{"/", "/app"} {
		uri = strings.TrimSuffix(uri, suffix)
	}

	log := util.NewLogger("tapo")

	// nosemgrep:go.lang.security.audit.crypto.use_of_weak_crypto.use-of-sha1
	h := sha1.New()
	_, err := h.Write([]byte(user))
	userhash := hex.EncodeToString(h.Sum(nil))

	conn := &Connection{
		log:             log,
		Helper:          request.NewHelper(log),
		URI:             fmt.Sprintf("%s/app", util.DefaultScheme(uri, "http")),
		EncodedUser:     base64.StdEncoding.EncodeToString([]byte(userhash)),
		EncodedPassword: base64.StdEncoding.EncodeToString([]byte(password)),
	}

	conn.Client.Timeout = Timeout

	return conn, err
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

	if err := d.CheckErrorCode(res.ErrorCode); err != nil {
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

// ExecCmd executes a Tapo api command and provides the response
func (d *Connection) ExecCmd(method string, enable bool) (*DeviceResponse, error) {
	// refresh session id
	if time.Since(d.updated) >= 600*time.Minute {
		if err := d.Login(); err != nil {
			return nil, err
		}

		d.updated = time.Now()
	}

	return d.ExecMethod(method, enable)
}

// CurrentPower provides current power consuption
func (d *Connection) CurrentPower() (float64, error) {
	resp, err := d.ExecCmd("get_energy_usage", false)
	if err != nil {
		return 0, err
	}

	return float64(resp.Result.Current_Power) / 1e3, nil
}

// ChargedEnergy collects the daily charged energy
func (d *Connection) ChargedEnergy() (float64, error) {
	resp, err := d.ExecCmd("get_energy_usage", false)
	if err != nil {
		return 0, err
	}

	if resp.Result.Today_Energy > d.lasttodayenergy {
		d.energy = d.energy + (resp.Result.Today_Energy - d.lasttodayenergy)
	}
	d.lasttodayenergy = resp.Result.Today_Energy

	return float64(d.energy) / 1000, nil
}

// DoSecureRequest executes a Tapo device request by encding the request and decoding its response.
func (d *Connection) DoSecureRequest(uri string, taporequest map[string]interface{}) (*DeviceResponse, error) {
	payload, err := json.Marshal(taporequest)
	if err != nil {
		return nil, err
	}

	d.log.TRACE.Printf("request: %s", string(payload))

	encryptedRequest, err := d.Cipher.Encrypt(payload)
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"method": "securePassthrough",
		"params": map[string]interface{}{
			"request": base64.StdEncoding.EncodeToString(encryptedRequest),
		},
	}

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), map[string]string{
		"Cookie": d.SessionID,
	})
	if err != nil {
		return nil, err
	}

	var res *DeviceResponse
	if err := d.DoJSON(req, &res); err != nil {
		return nil, err
	}

	// Login atempt in case of tapo switch connection hicups
	if res.ErrorCode == 9999 {
		if err := d.Login(); err != nil {
			return nil, err
		}

		if err := d.DoJSON(req, &res); err != nil {
			return nil, err
		}
	}

	if err := d.CheckErrorCode(res.ErrorCode); err != nil {
		return nil, err
	}

	decodedResponse, err := base64.StdEncoding.DecodeString(res.Result.Response)
	if err != nil {
		return nil, err
	}

	decryptedResponse, err := d.Cipher.Decrypt(decodedResponse)
	if err != nil {
		return nil, err
	}

	d.log.TRACE.Printf("decrypted result: %v", string(decryptedResponse))

	var deviceResp *DeviceResponse
	err = json.Unmarshal(decryptedResponse, &deviceResp)

	return deviceResp, err
}

// Tapo helper functions

func (d *Connection) CheckErrorCode(errorCode int) error {
	errorDesc := map[int]string{
		0:     "Success",
		9999:  "Login failed, invalid user or password",
		-1002: "Incorrect Request/Method",
		-1003: "JSON formatting error ",
		-1010: "Invalid Public Key Length",
		-1012: "Invalid terminalUUID",
		-1501: "Invalid Request or Credentials",
	}

	if errorCode != 0 {
		return fmt.Errorf("tapo error %d: %s", errorCode, errorDesc[errorCode])
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

	return pkcs7.Unpad(decryptedPayload, aes.BlockSize)
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
