package leapmotor

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"maps"
	"math/rand"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emmansun/gmsm/sm4"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/pkcs12"
)

// p12SM4Block is the SM4 cipher keyed with the APK-embedded key, used for PKCS#12 password derivation.
// key length is always 16 bytes; NewCipher only errors on wrong length.
var p12SM4Block, _ = sm4.NewCipher([]byte{0x42, 0x9c, 0xf4, 0x50, 0xef, 0x91, 0x7a, 0x98, 0x54, 0x33, 0x43, 0x0b, 0xcf, 0xed, 0x62, 0xac})

// p12MemoryEncode applies PKCS7 padding then SM4-ECB encryption block by block.
func p12MemoryEncode(data []byte) []byte {
	block := p12SM4Block
	padLen := 16 - len(data)%16
	padded := make([]byte, len(data)+padLen)
	copy(padded, data)
	for i := len(data); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}
	out := make([]byte, len(padded))
	for i := 0; i < len(padded); i += 16 {
		block.Encrypt(out[i:i+16], padded[i:i+16])
	}
	return out
}

// deriveP12Password derives the PKCS#12 certificate password from login response fields.
func deriveP12Password(accountID, uid string) string {
	h := md5.Sum([]byte(accountID))
	cn := hex.EncodeToString(h[:])
	cnEven := make([]byte, 0, len(cn)/2)
	for i := 0; i < len(cn); i += 2 {
		cnEven = append(cnEven, cn[i])
	}
	uidOdd := make([]byte, 0, len(uid)/2)
	for i := 1; i < len(uid); i += 2 {
		uidOdd = append(uidOdd, uid[i])
	}
	appInput := []byte(cn + string(cnEven) + string(uidOdd))
	digest := sha256.Sum256(appInput)
	encoded := p12MemoryEncode(digest[:])
	b64 := base64.StdEncoding.EncodeToString(encoded[:12])
	if len(b64) > 15 {
		return b64[:15]
	}
	return b64
}

type usernameClaims struct {
	jwt.RegisteredClaims
	Username string `json:"user_name"`
}

// extractSessionDeviceID extracts the session deviceId from the JWT token payload.
func extractSessionDeviceID(token string) *string {
	var claims usernameClaims
	if _, _, err := jwt.NewParser(jwt.WithoutClaimsValidation()).ParseUnverified(token, &claims); err != nil {
		return nil
	}
	parts := strings.Split(claims.Username, ",")
	if len(parts) >= 3 && parts[2] != "" {
		return &parts[2]
	}
	return nil
}

// deriveSignKey runs HKDF-SHA256 to produce the 32-byte HMAC signing key.
func deriveSignKey(ikm, salt, info string) ([]byte, error) {
	r := hkdf.New(sha256.New, []byte(ikm), []byte(salt), []byte(info))
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, err
	}
	return key, nil
}

// buildLoginHeaders constructs SHA256-signed headers for the login request.
func buildLoginHeaders(deviceID, username, password, lang string) map[string]string {
	nonce := strconv.Itoa(rand.Intn(9000000) + 100000)
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	signInput := lang + deviceType + deviceID + "1" + username + "0" + "1" + nonce + password + policyID + source + ts + appVersion
	sign := fmt.Sprintf("%x", sha256.Sum256([]byte(signInput)))
	return map[string]string{
		"Content-Type":   "application/x-www-form-urlencoded; charset=UTF-8",
		"acceptLanguage": lang,
		"channel":        channel,
		"deviceType":     deviceType,
		"X-P12_ENC_ALG":  p12EncAlg,
		"source":         source,
		"version":        appVersion,
		"nonce":          nonce,
		"deviceId":       deviceID,
		"timestamp":      ts,
		"sign":           sign,
	}
}

// buildSignedHeaders constructs HMAC-SHA256 signed headers for authenticated requests.
func buildSignedHeaders(signKey []byte, deviceID, vin, lang, userID, token string, bodyParams map[string]string) map[string]string {
	nonce := strconv.Itoa(rand.Intn(9000000) + 100000)
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	fields := map[string]string{
		"acceptLanguage": lang,
		"channel":        channel,
		"deviceId":       deviceID,
		"deviceType":     deviceType,
		"nonce":          nonce,
		"source":         source,
		"timestamp":      ts,
		"version":        appVersion,
	}
	if vin != "" {
		fields["vin"] = vin
	}
	maps.Copy(fields, bodyParams)
	keys := slices.Collect(maps.Keys(fields))
	slices.Sort(keys)
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(fields[k])
	}
	mac := hmac.New(sha256.New, signKey)
	mac.Write([]byte(sb.String()))
	sign := hex.EncodeToString(mac.Sum(nil))
	return map[string]string{
		"Content-Type":   "application/x-www-form-urlencoded",
		"acceptLanguage": lang,
		"channel":        channel,
		"deviceType":     deviceType,
		"X-P12_ENC_ALG":  p12EncAlg,
		"source":         source,
		"version":        appVersion,
		"nonce":          nonce,
		"deviceId":       deviceID,
		"timestamp":      ts,
		"sign":           sign,
		"userId":         userID,
		"token":          token,
	}
}

// newMTLSClient creates an http.Client with optional client cert and TLS verification disabled.
// Leapmotor's API servers use self-signed certificates.
func newMTLSClient(cert *tls.Certificate) *http.Client {
	tlsCfg := &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	if cert != nil {
		tlsCfg.Certificates = []tls.Certificate{*cert}
	}
	return &http.Client{
		Transport: &http.Transport{TLSClientConfig: tlsCfg},
		Timeout:   30 * time.Second,
	}
}

// loadAccountCert decodes a PKCS#12 bundle and returns a tls.Certificate.
func loadAccountCert(p12Data []byte, password string) (tls.Certificate, error) {
	priv, cert, err := pkcs12.Decode(p12Data, password)
	if err != nil {
		return tls.Certificate{}, err
	}
	if cert == nil || priv == nil {
		return tls.Certificate{}, fmt.Errorf("pkcs12: missing cert or key")
	}
	return tls.Certificate{
		Certificate: [][]byte{cert.Raw},
		PrivateKey:  priv,
		Leaf:        cert,
	}, nil
}

// Identity manages Leapmotor session state (tokens, signing key, account mTLS cert).
const (
	defaultOperpwdKey = "f1cf0c025baec0e2"
	defaultOperpwdIV  = "6b6a1fe94e133fd7"
)

type Identity struct {
	mu         sync.Mutex
	log        *util.Logger
	appCert    tls.Certificate
	username   string
	password   string
	pin        string
	certSynced bool
	// mutable session state (protected by mu)
	deviceID   string
	token      string
	userID     string
	refreshTok string
	signKey    []byte
	acctClient *http.Client
}

// NewIdentity parses the app certificate PEM bytes and returns an unauthenticated Identity.
// Call Login() before making API calls. pin is optional; pass "" to disable charge control.
func NewIdentity(log *util.Logger, certPEM, keyPEM []byte, username, password, pin string) (*Identity, error) {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("load app cert: %w", err)
	}
	h := sha256.Sum256([]byte(username))
	return &Identity{
		log:      log,
		appCert:  cert,
		username: username,
		password: password,
		pin:      pin,
		deviceID: hex.EncodeToString(h[:16]),
	}, nil
}

// HasPin reports whether a vehicle PIN is configured for charge control.
func (id *Identity) HasPin() bool { return id.pin != "" }

// EncryptedPin returns the AES-128-CBC encrypted vehicle PIN using the current session token.
func (id *Identity) EncryptedPin() (string, error) {
	if id.pin == "" {
		return "", fmt.Errorf("no vehicle PIN configured")
	}
	id.mu.Lock()
	token := id.token
	id.mu.Unlock()

	keyText, ivText := defaultOperpwdKey, defaultOperpwdIV
	if len(token) >= 64 {
		kh := md5.Sum([]byte(token[:32]))
		ih := md5.Sum([]byte(token[32:64]))
		keyText = hex.EncodeToString(kh[:])[8:24]
		ivText = hex.EncodeToString(ih[:])[8:24]
	}

	block, err := aes.NewCipher([]byte(keyText))
	if err != nil {
		return "", err
	}
	pinBytes := []byte(id.pin)
	padLen := aes.BlockSize - len(pinBytes)%aes.BlockSize
	padded := append(pinBytes, bytes.Repeat([]byte{byte(padLen)}, padLen)...)
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, []byte(ivText)).CryptBlocks(ciphertext, padded)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// CertSync registers the app certificate with the Leapmotor server (required before remote control).
// Only called once per session; subsequent calls are no-ops.
func (id *Identity) CertSync() error {
	id.mu.Lock()
	if id.certSynced {
		id.mu.Unlock()
		return nil
	}
	token, userID, deviceID, signKey := id.token, id.userID, id.deviceID, id.signKey
	appCert := id.appCert
	id.mu.Unlock()

	appClient := newMTLSClient(&appCert)
	headers := buildSignedHeaders(signKey, deviceID, "", defaultLang, userID, token, nil)
	if _, err := apiPost(appClient, BaseURL+"/carownerservice/oversea/vehicle/v1/cert/sync", headers, ""); err != nil {
		return fmt.Errorf("cert sync: %w", err)
	}

	id.mu.Lock()
	id.certSynced = true
	id.mu.Unlock()
	return nil
}

// Login performs a full authentication and loads the account certificate.
func (id *Identity) Login() error {
	id.mu.Lock()
	defer id.mu.Unlock()
	return id.login()
}

func (id *Identity) login() error {
	appClient := newMTLSClient(&id.appCert)
	headers := buildLoginHeaders(id.deviceID, id.username, id.password, defaultLang)
	body := url.Values{
		"isRecoverAcct": {"0"},
		"password":      {id.password},
		"policyId":      {policyID},
		"loginMethod":   {"1"},
		"email":         {id.username},
	}.Encode()

	data, err := postAndParse[LoginResponse](appClient, BaseURL+"/carownerservice/oversea/acct/v1/login", headers, body)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	id.token = data.Token
	id.refreshTok = data.RefreshToken
	id.userID = data.ID.String()
	if newDeviceID := extractSessionDeviceID(data.Token); newDeviceID != nil {
		id.deviceID = *newDeviceID
	}

	signKey, err := deriveSignKey(data.SignIkm, data.SignSalt, data.SignInfo)
	if err != nil {
		return fmt.Errorf("derive sign key: %w", err)
	}
	id.signKey = signKey

	p12Bytes, err := base64.StdEncoding.DecodeString(data.Base64Cert)
	if err != nil {
		return fmt.Errorf("decode base64cert: %w", err)
	}
	pwd := deriveP12Password(id.userID, data.UID)
	acctCert, err := loadAccountCert(p12Bytes, pwd)
	if err != nil {
		return fmt.Errorf("load account cert (derived password): %w", err)
	}
	id.acctClient = newMTLSClient(&acctCert)

	prefix := "leapmotor." + id.deviceID + "."
	settings.SetString(prefix+"token", id.token)
	settings.SetString(prefix+"refresh", id.refreshTok)
	settings.SetString(prefix+"userid", id.userID)
	settings.SetString(prefix+"signkey", hex.EncodeToString(id.signKey))
	settings.SetString(prefix+"p12", data.Base64Cert)
	settings.SetString(prefix+"p12pwd", pwd)
	return nil
}

// TryRestore loads a previously persisted session from the DB.
// Returns an error if no valid cached session exists.
func (id *Identity) TryRestore() error {
	id.mu.Lock()
	defer id.mu.Unlock()

	prefix := "leapmotor." + id.deviceID + "."
	token, err1 := settings.String(prefix + "token")
	refreshTok, err2 := settings.String(prefix + "refresh")
	userID, err3 := settings.String(prefix + "userid")
	signKeyHex, err4 := settings.String(prefix + "signkey")
	p12B64, err5 := settings.String(prefix + "p12")
	p12Pwd, err6 := settings.String(prefix + "p12pwd")
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || err6 != nil {
		return fmt.Errorf("no cached session")
	}

	signKey, err := hex.DecodeString(signKeyHex)
	if err != nil {
		return fmt.Errorf("decode sign key: %w", err)
	}
	p12Bytes, err := base64.StdEncoding.DecodeString(p12B64)
	if err != nil {
		return fmt.Errorf("decode p12: %w", err)
	}
	acctCert, err := loadAccountCert(p12Bytes, p12Pwd)
	if err != nil {
		return fmt.Errorf("load cached account cert: %w", err)
	}

	id.token = token
	id.refreshTok = refreshTok
	id.userID = userID
	id.signKey = signKey
	id.acctClient = newMTLSClient(&acctCert)
	return nil
}

// Refresh refreshes the access token; falls back to a full login on failure.
func (id *Identity) Refresh() error {
	id.mu.Lock()
	defer id.mu.Unlock()
	if id.refreshTok == "" {
		return id.login()
	}
	bodyParams := map[string]string{"refreshToken": id.refreshTok}
	headers := buildSignedHeaders(id.signKey, id.deviceID, "", defaultLang, id.userID, id.token, bodyParams)
	body := "refreshToken=" + url.QueryEscape(id.refreshTok)

	type refreshData struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refreshToken"`
	}
	d, err := postAndParse[refreshData](id.acctClient, BaseURL+"/carownerservice/oversea/acct/v1/token/refresh", headers, body)
	if err != nil {
		id.log.DEBUG.Printf("token refresh failed: %v; re-logging in", err)
		return id.login()
	}
	id.token = d.Token
	id.refreshTok = d.RefreshToken
	return nil
}

// Session returns a snapshot of credentials for use in a single request.
func (id *Identity) Session() (acctClient *http.Client, token, userID, deviceID string, signKey []byte) {
	id.mu.Lock()
	defer id.mu.Unlock()
	return id.acctClient, id.token, id.userID, id.deviceID, id.signKey
}
