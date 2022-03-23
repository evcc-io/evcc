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

func (d *Connection) DoRequest(uri string, request []byte) ([]byte, error) {
	securedReq, _ := json.Marshal(map[string]interface{}{
		"method": "securePassthrough",
		"params": map[string]interface{}{
			"request": base64.StdEncoding.EncodeToString(d.Cipher.Encrypt(request)),
		},
	})

	fmt.Printf("securedReq:\n%s\n", string(securedReq))

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
		return nil, err
	}

	encryptedResponse, _ := base64.StdEncoding.DecodeString(jsonResp.Result.Response)

	return d.Cipher.Decrypt(encryptedResponse), nil
}

func (d *Connection) CheckErrorCode(errorCode int) error {
	if errorCode != 0 {
		return errors.New(fmt.Sprintf("Got error code %d", errorCode))
	}

	return nil
}

func (d *Connection) Handshake() (err error) {
	privKey, pubKey := GenerateRSAKeys()

	pubPEM := DumpRSAPEM(pubKey)
	req, _ := json.Marshal(map[string]interface{}{
		"method": "handshake",
		"params": map[string]interface{}{
			"key":             string(pubPEM),
			"requestTimeMils": 0,
		},
	})

	resp, err := http.Post(d.URI, "application/json", bytes.NewBuffer(req))
	if err != nil {
		return
	}

	defer resp.Body.Close()

	var jsonResp DeviceResponse
	json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err = d.CheckErrorCode(jsonResp.ErrorCode); err != nil {
		return
	}

	encryptedEncryptionKey, _ := base64.StdEncoding.DecodeString(jsonResp.Result.Key)
	encryptionKey, _ := rsa.DecryptPKCS1v15(rand.Reader, privKey, encryptedEncryptionKey)
	d.Cipher = &ConnectionCipher{
		Key: encryptionKey[:16],
		Iv:  encryptionKey[16:],
	}

	d.SessionID = strings.Split(resp.Header.Get("Set-Cookie"), ";")[0]

	return
}

func (d *Connection) Login() (err error) {
	if d.Cipher == nil {
		return errors.New("Handshake was not performed")
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"method": "login_device",
		"params": map[string]interface{}{
			"username": d.EncodedUser,
			"password": d.EncodedPassword,
		},
	})
	fmt.Printf("payload:\n%s\n", string(payload))

	payload, err = d.DoRequest(d.URI, payload)
	if err != nil {
		return
	}

	var jsonResp DeviceResponse
	json.NewDecoder(bytes.NewBuffer(payload)).Decode(&jsonResp)
	if err = d.CheckErrorCode(jsonResp.ErrorCode); err != nil {
		return
	}

	d.Token = &jsonResp.Result.Token
	return
}

func (d *Connection) GetDeviceInfo() (*DeviceInfo, error) {
	if d.Token == nil {
		return nil, errors.New("Login was not performed")
	}

	req, _ := json.Marshal(map[string]interface{}{
		"method": "get_device_info",
	})

	resp, err := d.DoRequest(fmt.Sprintf("%s?token=%s", d.URI, *d.Token), req)
	if err != nil {
		return nil, err
	}

	status := &DeviceInfo{}

	json.NewDecoder(bytes.NewBuffer(resp)).Decode(status)
	if err = d.CheckErrorCode(status.ErrorCode); err != nil {
		return nil, err
	}

	nicknameEncoded, _ := base64.StdEncoding.DecodeString(status.Result.Nickname)
	status.Result.Nickname = string(nicknameEncoded)

	SSIDEncoded, _ := base64.StdEncoding.DecodeString(status.Result.SSID)
	status.Result.SSID = string(SSIDEncoded)

	return status, nil
}

func (d *Connection) ExecMethod(method string) (map[string]interface{}, error) {
	if method == "get_device_info" {
	}
	return nil, nil
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
