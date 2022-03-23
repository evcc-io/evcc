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

	"github.com/mergermarket/go-pkcs7"
)

const Timeout = time.Second * 15

func New(ip, email, password string) *Device {
	h := sha1.New()
	h.Write([]byte(email))
	return &Device{
		ip:              ip,
		encodedEmail:    base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(h.Sum(nil)))),
		encodedPassword: base64.StdEncoding.EncodeToString([]byte(password)),
		client:          &http.Client{Timeout: Timeout},
	}
}

func (d *Device) GetURL() string {
	if d.token == nil {
		return fmt.Sprintf("http://%s/app", d.ip)
	} else {
		return fmt.Sprintf("http://%s/app?token=%s", d.ip, *d.token)
	}
}

func (d *Device) DoRequest(payload []byte) ([]byte, error) {
	securedPayload, _ := json.Marshal(map[string]interface{}{
		"method": "securePassthrough",
		"params": map[string]interface{}{
			"request": base64.StdEncoding.EncodeToString(d.cipher.Encrypt(payload)),
		},
	})

	fmt.Printf("securedPayload:\n%s\n", string(securedPayload))

	req, _ := http.NewRequest("POST", d.GetURL(), bytes.NewBuffer(securedPayload))
	req.Header.Set("Cookie", d.sessionID)
	req.Close = true

	resp, err := d.client.Do(req)
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

	return d.cipher.Decrypt(encryptedResponse), nil
}

func (d *Device) CheckErrorCode(errorCode int) error {
	if errorCode != 0 {
		return errors.New(fmt.Sprintf("Got error code %d", errorCode))
	}

	return nil
}

func (d *Device) Handshake() (err error) {
	privKey, pubKey := GenerateRSAKeys()

	pubPEM := DumpRSAPEM(pubKey)
	payload, _ := json.Marshal(map[string]interface{}{
		"method": "handshake",
		"params": map[string]interface{}{
			"key":             string(pubPEM),
			"requestTimeMils": 0,
		},
	})

	resp, err := http.Post(d.GetURL(), "application/json", bytes.NewBuffer(payload))
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
	d.cipher = &DeviceCipher{
		key: encryptionKey[:16],
		iv:  encryptionKey[16:],
	}

	d.sessionID = strings.Split(resp.Header.Get("Set-Cookie"), ";")[0]

	return
}

func (d *Device) Login() (err error) {
	if d.cipher == nil {
		return errors.New("Handshake was not performed")
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"method": "login_device",
		"params": map[string]interface{}{
			"username": d.encodedEmail,
			"password": d.encodedPassword,
		},
	})
	fmt.Printf("payload:\n%s\n", string(payload))

	payload, err = d.DoRequest(payload)
	if err != nil {
		return
	}

	var jsonResp DeviceResponse
	json.NewDecoder(bytes.NewBuffer(payload)).Decode(&jsonResp)
	if err = d.CheckErrorCode(jsonResp.ErrorCode); err != nil {
		return
	}

	d.token = &jsonResp.Result.Token
	return
}

func (d *Device) GetDeviceInfo() (*DeviceInfo, error) {
	if d.token == nil {
		return nil, errors.New("Login was not performed")
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"method": "get_device_info",
	})

	payload, err := d.DoRequest(payload)
	if err != nil {
		return nil, err
	}

	status := &DeviceInfo{}

	json.NewDecoder(bytes.NewBuffer(payload)).Decode(status)
	if err = d.CheckErrorCode(status.ErrorCode); err != nil {
		return nil, err
	}

	nicknameEncoded, _ := base64.StdEncoding.DecodeString(status.Result.Nickname)
	status.Result.Nickname = string(nicknameEncoded)

	SSIDEncoded, _ := base64.StdEncoding.DecodeString(status.Result.SSID)
	status.Result.SSID = string(SSIDEncoded)

	return status, nil
}

func (c *DeviceCipher) Encrypt(payload []byte) []byte {
	block, _ := aes.NewCipher(c.key)
	encrypter := cipher.NewCBCEncrypter(block, c.iv)

	paddedPayload, _ := pkcs7.Pad(payload, aes.BlockSize)
	encryptedPayload := make([]byte, len(paddedPayload))
	encrypter.CryptBlocks(encryptedPayload, paddedPayload)

	return encryptedPayload
}

func (c *DeviceCipher) Decrypt(payload []byte) []byte {
	block, _ := aes.NewCipher(c.key)
	encrypter := cipher.NewCBCDecrypter(block, c.iv)

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
