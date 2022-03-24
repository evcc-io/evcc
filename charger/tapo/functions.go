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
	"github.com/mergermarket/go-pkcs7"
)

const Timeout = time.Second * 15

func NewConnection(uri, user, password string) *Connection {

	settings := &Settings{
		URI:      fmt.Sprintf("%s/app", util.DefaultScheme(uri, "http")),
		User:     user,
		Password: password,
	}

	h := sha1.New()
	h.Write([]byte(user))

	tapo := &Connection{
		Settings:        settings,
		EncodedUser:     base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(h.Sum(nil)))),
		EncodedPassword: base64.StdEncoding.EncodeToString([]byte(password)),
		Client:          &http.Client{Timeout: Timeout},
	}

	return tapo
}

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

	res, err := d.DoRequest(d.URI, req)
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

	res, err := d.DoRequest(fmt.Sprintf("%s?token=%s", d.URI, *d.Token), req)
	if err != nil {
		return nil, err
	}

	tapoResp := &DeviceResponse{}
	json.NewDecoder(bytes.NewBuffer(res)).Decode(tapoResp)
	if err = d.CheckErrorCode(tapoResp.ErrorCode); err != nil {
		return tapoResp, err
	}

	if method == "get_device_info" {
		tapoResp.Result.Nickname = base64Decode(tapoResp.Result.Nickname)
		tapoResp.Result.SSID = base64Decode(tapoResp.Result.SSID)
	}

	return tapoResp, nil
}

func (d *Connection) DoRequest(uri string, request []byte) ([]byte, error) {
	securedReq, _ := json.Marshal(map[string]interface{}{
		"method": "securePassthrough",
		"params": map[string]interface{}{
			"request": base64.StdEncoding.EncodeToString(d.Cipher.Encrypt(request)),
		},
	})

	req, _ := http.NewRequest("POST", uri, bytes.NewBuffer(securedReq))
	req.Header.Set("Cookie", d.SessionID)
	req.Close = true

	resp, err := d.Client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var jsonResp DeviceResponse
	json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err = d.CheckErrorCode(jsonResp.ErrorCode); err != nil {
		return []byte(fmt.Sprintf("{\"error_code\":%v}", jsonResp.ErrorCode)), err
	}

	encryptedResponse, _ := base64.StdEncoding.DecodeString(jsonResp.Result.Response)

	return d.Cipher.Decrypt(encryptedResponse), nil
}

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
