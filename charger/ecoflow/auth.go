// Ported from shuette42/ecoflow-energy-ha (MIT), see the package documentation.

package ecoflow

import (
	"bytes"
	"cmp"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// apiBases are the EcoFlow app API hosts, tried in order. The one that accepts
// the login is used for the certificate as well.
var apiBases = []string{"https://api-e.ecoflow.com", "https://api.ecoflow.com"}

// certIV is the fixed initialisation vector of the EcoFlow portal bundle.
const certIV = "ojsajkqjwk1w2dfg"

// Session holds what the MQTT layer needs after a successful login.
type Session struct {
	UserID   string
	Account  string // MQTT username
	Password string // MQTT password
	Base     string // API host that accepted the login
	Token    string // app API token
}

type response struct {
	Code    any    `json:"code"`
	Message string `json:"message"`
}

func (r response) error() error {
	if code := fmt.Sprint(r.Code); code != "0" {
		return fmt.Errorf("code %s: %s", code, r.Message)
	}
	return nil
}

// Login authenticates like the EcoFlow app and fetches the MQTT credentials.
// The token and the MQTT credentials are added to the logger's redaction list.
func Login(log *util.Logger, email, password string) (*Session, error) {
	encoded := base64.StdEncoding.EncodeToString([]byte(password))
	log.Redact(email, password, encoded)

	// request.Helper would trace the token and user ID before they are redacted
	client := &request.Helper{Client: &http.Client{Timeout: request.Timeout, Transport: transport.Default()}}

	var errs []error
	for _, base := range apiBases {
		token, userID, err := login(client, base, email, encoded)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", base, err))
			continue
		}
		log.Redact(token)

		account, mqttPassword, err := certificate(client, base, token)
		if err != nil {
			return nil, fmt.Errorf("certificate: %w", err)
		}
		log.Redact(account, mqttPassword)
		log.DEBUG.Printf("logged in at %s", base)

		return &Session{UserID: userID, Account: account, Password: mqttPassword, Base: base, Token: token}, nil
	}

	return nil, fmt.Errorf("login: %w", errors.Join(errs...))
}

func login(client *request.Helper, base, email, encodedPassword string) (string, string, error) {
	data := map[string]string{
		"email":    email,
		"password": encodedPassword,
		"scene":    "IOT_APP",
		"userType": "ECOFLOW",
	}

	req, err := request.New(http.MethodPost, base+"/auth/login", request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return "", "", err
	}

	var res struct {
		response
		Data struct {
			Token string `json:"token"`
			User  struct {
				UserID any `json:"userId"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := client.DoJSON(req, &res); err != nil {
		return "", "", err
	}
	if err := res.error(); err != nil {
		return "", "", err
	}

	userID := fmt.Sprint(res.Data.User.UserID)
	if res.Data.Token == "" || res.Data.User.UserID == nil || userID == "" {
		return "", "", errors.New("missing token or user id")
	}

	return res.Data.Token, userID, nil
}

// certificate retrieves and decrypts the MQTT credentials.
func certificate(client *request.Helper, base, token string) (string, string, error) {
	req, err := request.New(http.MethodGet, base+"/iot-auth/enterprise-development/user/certification", nil,
		request.AcceptJSON, map[string]string{"Authorization": "Bearer " + token})
	if err != nil {
		return "", "", err
	}

	var res struct {
		response
		Data string `json:"data"`
	}
	if err := client.DoJSON(req, &res); err != nil {
		return "", "", err
	}
	if err := res.error(); err != nil {
		return "", "", err
	}

	plain, err := decryptCertificate(token, res.Data)
	if err != nil {
		return "", "", err
	}

	// the app session returns userName/password, the developer API certificateAccount/certificatePassword
	var cert struct {
		UserName            string `json:"userName"`
		Password            string `json:"password"`
		CertificateAccount  string `json:"certificateAccount"`
		CertificatePassword string `json:"certificatePassword"`
	}
	if err := json.Unmarshal(plain, &cert); err != nil {
		return "", "", errors.New("decrypted certificate is not JSON")
	}

	account, password := cmp.Or(cert.UserName, cert.CertificateAccount), cmp.Or(cert.Password, cert.CertificatePassword)
	if account == "" || password == "" {
		return "", "", errors.New("no MQTT credentials in certificate")
	}

	return account, password, nil
}

// decryptCertificate decrypts the AES-CFB ciphertext keyed by the SHA-256 of
// the token.
func decryptCertificate(token, blob string) ([]byte, error) {
	ct, err := base64.StdEncoding.DecodeString(blob)
	if err != nil {
		return nil, fmt.Errorf("base64: %w", err)
	}
	if len(ct) == 0 || len(ct)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext length %d is not a block multiple", len(ct))
	}

	key := sha256.Sum256([]byte(token))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	plain := make([]byte, len(ct))
	cipher.NewCFBDecrypter(block, []byte(certIV)).XORKeyStream(plain, ct)
	return stripPKCS7(plain), nil
}

// stripPKCS7 removes padding when it is well formed and leaves the input
// untouched otherwise, because the server does not always pad.
func stripPKCS7(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	n := int(b[len(b)-1])
	if n <= 0 || n > aes.BlockSize || n > len(b) {
		return b
	}
	if !bytes.Equal(b[len(b)-n:], bytes.Repeat([]byte{byte(n)}, n)) {
		return b
	}
	return b[:len(b)-n]
}
