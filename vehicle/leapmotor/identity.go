package leapmotor

import (
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emmansun/gmsm/sm4"
	"github.com/evcc-io/evcc/util"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/pkcs12"
)

// p12SM4Block is the SM4 cipher keyed with the APK-embedded key, used for PKCS#12 password derivation.
var p12SM4Block cipher.Block

func init() {
	// key length is always 16 bytes; NewCipher only errors on wrong length
	p12SM4Block, _ = sm4.NewCipher([]byte{0x42, 0x9c, 0xf4, 0x50, 0xef, 0x91, 0x7a, 0x98, 0x54, 0x33, 0x43, 0x0b, 0xcf, 0xed, 0x62, 0xac})
}

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

// deriveSessionDeviceID extracts the session deviceId from the JWT token payload.
func deriveSessionDeviceID(token, fallback string) string {
	var claims jwt.MapClaims
	if _, _, err := jwt.NewParser(jwt.WithoutClaimsValidation()).ParseUnverified(token, &claims); err != nil {
		return fallback
	}
	userName, _ := claims["user_name"].(string)
	parts := strings.Split(userName, ",")
	if len(parts) >= 4 && parts[2] != "" {
		return parts[2]
	}
	return fallback
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
func buildSignedHeaders(signKey []byte, deviceID, vin, lang string, bodyParams map[string]string) map[string]string {
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
	for k, v := range bodyParams {
		fields[k] = v
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(fields[k])
	}
	mac := hmac.New(sha256.New, signKey)
	mac.Write([]byte(sb.String()))
	sign := fmt.Sprintf("%x", mac.Sum(nil))
	return map[string]string{
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

// addAuthHeaders merges Content-Type, userId and token into the provided header map.
func addAuthHeaders(headers map[string]string, userID, token string) {
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	headers["userId"] = userID
	headers["token"] = token
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
type Identity struct {
	mu       sync.Mutex
	log      *util.Logger
	appCert  tls.Certificate
	username string
	password string
	// mutable session state (protected by mu)
	deviceID   string
	token      string
	userID     string
	refreshTok string
	signKey    []byte
	acctClient *http.Client
}

// NewIdentity parses the app certificate PEM bytes and returns an unauthenticated Identity.
// Call Login() before making API calls.
func NewIdentity(log *util.Logger, certPEM, keyPEM []byte, username, password string) (*Identity, error) {
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
		deviceID: hex.EncodeToString(h[:16]),
	}, nil
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
	id.deviceID = deriveSessionDeviceID(data.Token, id.deviceID)

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
	headers := buildSignedHeaders(id.signKey, id.deviceID, "", defaultLang, bodyParams)
	addAuthHeaders(headers, id.userID, id.token)
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
