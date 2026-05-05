package leapmotor

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/bits"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/pkcs12"
)

// SM4 cipher (ECB, fixed round keys from APK) — used only for PKCS#12 password derivation.

var sm4SBox = [256]byte{
	0xD6, 0x90, 0xE9, 0xFE, 0xCC, 0xE1, 0x3D, 0xB7, 0x16, 0xB6, 0x14, 0xC2, 0x28, 0xFB, 0x2C, 0x05,
	0x2B, 0x67, 0x9A, 0x76, 0x2A, 0xBE, 0x04, 0xC3, 0xAA, 0x44, 0x13, 0x26, 0x49, 0x86, 0x06, 0x99,
	0x9C, 0x42, 0x50, 0xF4, 0x91, 0xEF, 0x98, 0x7A, 0x33, 0x54, 0x0B, 0x43, 0xED, 0xCF, 0xAC, 0x62,
	0xE4, 0xB3, 0x1C, 0xA9, 0xC9, 0x08, 0xE8, 0x95, 0x80, 0xDF, 0x94, 0xFA, 0x75, 0x8F, 0x3F, 0xA6,
	0x47, 0x07, 0xA7, 0xFC, 0xF3, 0x73, 0x17, 0xBA, 0x83, 0x59, 0x3C, 0x19, 0xE6, 0x85, 0x4F, 0xA8,
	0x68, 0x6B, 0x81, 0xB2, 0x71, 0x64, 0xDA, 0x8B, 0xF8, 0xEB, 0x0F, 0x4B, 0x70, 0x56, 0x9D, 0x35,
	0x1E, 0x24, 0x0E, 0x5E, 0x63, 0x58, 0xD1, 0xA2, 0x25, 0x22, 0x7C, 0x3B, 0x01, 0x21, 0x78, 0x87,
	0xD4, 0x00, 0x46, 0x57, 0x9F, 0xD3, 0x27, 0x52, 0x4C, 0x36, 0x02, 0xE7, 0xA0, 0xC4, 0xC8, 0x9E,
	0xEA, 0xBF, 0x8A, 0xD2, 0x40, 0xC7, 0x38, 0xB5, 0xA3, 0xF7, 0xF2, 0xCE, 0xF9, 0x61, 0x15, 0xA1,
	0xE0, 0xAE, 0x5D, 0xA4, 0x9B, 0x34, 0x1A, 0x55, 0xAD, 0x93, 0x32, 0x30, 0xF5, 0x8C, 0xB1, 0xE3,
	0x1D, 0xF6, 0xE2, 0x2E, 0x82, 0x66, 0xCA, 0x60, 0xC0, 0x29, 0x23, 0xAB, 0x0D, 0x53, 0x4E, 0x6F,
	0xD5, 0xDB, 0x37, 0x45, 0xDE, 0xFD, 0x8E, 0x2F, 0x03, 0xFF, 0x6A, 0x72, 0x6D, 0x6C, 0x5B, 0x51,
	0x8D, 0x1B, 0xAF, 0x92, 0xBB, 0xDD, 0xBC, 0x7F, 0x11, 0xD9, 0x5C, 0x41, 0x1F, 0x10, 0x5A, 0xD8,
	0x0A, 0xC1, 0x31, 0x88, 0xA5, 0xCD, 0x7B, 0xBD, 0x2D, 0x74, 0xD0, 0x12, 0xB8, 0xE5, 0xB4, 0xB0,
	0x89, 0x69, 0x97, 0x4A, 0x0C, 0x96, 0x77, 0x7E, 0x65, 0xB9, 0xF1, 0x09, 0xC5, 0x6E, 0xC6, 0x84,
	0x18, 0xF0, 0x7D, 0xEC, 0x3A, 0xDC, 0x4D, 0x20, 0x79, 0xEE, 0x5F, 0x3E, 0xD7, 0xCB, 0x39, 0x48,
}

var sm4RoundKeys = [32]uint32{
	0x818FA553, 0xEBA3318D, 0x5FC3C93A, 0xBD1DADD9,
	0xBB61CAB9, 0x000FD7EA, 0xDC6E0166, 0xDA937279,
	0x607EE786, 0xB548754C, 0x107330E4, 0xEA17C186,
	0x0F56F74B, 0xB21E443C, 0xE1210FE2, 0x009995C8,
	0xE7529A48, 0x6EF474F6, 0x2AB06DF6, 0x43B11BE8,
	0x359D4A14, 0xC29E2CDE, 0x30CF6A3E, 0x79D1C806,
	0x7C502387, 0xAAAB9BC6, 0xF0FE744B, 0x1CAFC872,
	0x95A9D075, 0x88070D58, 0x22800475, 0x8391938B,
}

func sm4EncryptBlock(block [16]byte) [16]byte {
	x0 := binary.BigEndian.Uint32(block[0:4])
	x1 := binary.BigEndian.Uint32(block[4:8])
	x2 := binary.BigEndian.Uint32(block[8:12])
	x3 := binary.BigEndian.Uint32(block[12:16])
	for _, rk := range sm4RoundKeys {
		t := x1 ^ x2 ^ x3 ^ rk
		b := uint32(sm4SBox[t>>24])<<24 |
			uint32(sm4SBox[(t>>16)&0xFF])<<16 |
			uint32(sm4SBox[(t>>8)&0xFF])<<8 |
			uint32(sm4SBox[t&0xFF])
		newX := x0 ^ b ^ bits.RotateLeft32(b, 2) ^ bits.RotateLeft32(b, 10) ^ bits.RotateLeft32(b, 18) ^ bits.RotateLeft32(b, 24)
		x0, x1, x2, x3 = x1, x2, x3, newX
	}
	var out [16]byte
	binary.BigEndian.PutUint32(out[0:4], x3)
	binary.BigEndian.PutUint32(out[4:8], x2)
	binary.BigEndian.PutUint32(out[8:12], x1)
	binary.BigEndian.PutUint32(out[12:16], x0)
	return out
}

// p12MemoryEncode applies PKCS7 padding then SM4-ECB encryption block by block.
func p12MemoryEncode(data []byte) []byte {
	padLen := 16 - len(data)%16
	padded := make([]byte, len(data)+padLen)
	copy(padded, data)
	for i := len(data); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}
	out := make([]byte, len(padded))
	for i := 0; i < len(padded); i += 16 {
		var block [16]byte
		copy(block[:], padded[i:i+16])
		enc := sm4EncryptBlock(block)
		copy(out[i:], enc[:])
	}
	return out
}

// deriveP12Password derives the PKCS#12 certificate password from login response fields.
func deriveP12Password(accountID, uid string) string {
	h := md5.Sum([]byte(accountID))
	cn := fmt.Sprintf("%x", h)
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
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return fallback
	}
	payload := parts[1]
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}
	raw, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return fallback
	}
	var claims map[string]any
	if json.Unmarshal(raw, &claims) != nil {
		return fallback
	}
	userName, _ := claims["user_name"].(string)
	claimParts := strings.Split(userName, ",")
	if len(claimParts) >= 4 && claimParts[2] != "" {
		return claimParts[2]
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

// NewIdentity loads the app certificate and returns an unauthenticated Identity.
// Call Login() before making API calls.
func NewIdentity(log *util.Logger, appCertFile, appKeyFile, username, password string) (*Identity, error) {
	cert, err := tls.LoadX509KeyPair(appCertFile, appKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load app cert: %w", err)
	}
	b := make([]byte, 16)
	rand.Read(b) //nolint:errcheck
	return &Identity{
		log:      log,
		appCert:  cert,
		username: username,
		password: password,
		deviceID: fmt.Sprintf("%x", b),
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

	respBody, err := apiPost(appClient, BaseURL+"/carownerservice/oversea/acct/v1/login", headers, body)
	if err != nil {
		return fmt.Errorf("login request: %w", err)
	}

	data, err := parseEnvelope[LoginResponse](respBody)
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

	respBody, err := apiPost(id.acctClient, BaseURL+"/carownerservice/oversea/acct/v1/token/refresh", headers, body)
	if err != nil {
		id.log.DEBUG.Printf("token refresh request failed: %v; re-logging in", err)
		return id.login()
	}

	type refreshData struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refreshToken"`
	}
	d, err := parseEnvelope[refreshData](respBody)
	if err != nil {
		id.log.DEBUG.Printf("token refresh parse failed: %v; re-logging in", err)
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
