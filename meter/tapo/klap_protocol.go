package tapo

// Tapo KLAP protocol, it replicates the code at
// https://github.com/python-kasa/python-kasa/pull/509

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/netip"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func NewKlapSession(l *log.Logger) *KlapSession {
	return &KlapSession{
		log: l,
	}
}

type KlapSession struct {
	log         *log.Logger
	addr        netip.Addr
	SessionID   string
	Expiry      time.Time
	LocalSeed   []byte
	RemoteSeed  []byte
	UserHash    []byte
	key         []byte
	sig         []byte
	iv          []byte
	seq         int32
	initialized bool
}

func (s *KlapSession) Addr() netip.Addr {
	return s.addr
}

func (s *KlapSession) secretBytes() []byte {
	ret := append(s.LocalSeed, s.RemoteSeed...)
	return append(ret, s.UserHash...)
}

func (s *KlapSession) getKey() []byte {
	if s.key == nil {
		bytesToHash := append([]byte("lsk"), s.secretBytes()...)
		key := sha256.Sum256(bytesToHash)
		s.key = key[:16]
	}
	return s.key
}

func (s *KlapSession) getSignature() []byte {
	if s.sig == nil {
		bytesToHash := append([]byte("ldk"), s.secretBytes()...)
		sig := sha256.Sum256(bytesToHash)
		s.sig = sig[:28]
	}
	return s.sig
}

func (s *KlapSession) getIV() []byte {
	if s.iv == nil {
		bytesToHash := append([]byte("iv"), s.secretBytes()...)
		hash := sha256.Sum256(bytesToHash)
		s.iv = append(hash[:12], hash[len(hash)-4:]...)
	}
	return s.iv
}

func (s *KlapSession) encrypt(data []byte) ([]byte, int32, error) {
	s.log.Printf("Plaintext: %s", data)
	key := s.getKey()
	if !s.initialized {
		s.iv = s.getIV()
		s.seq = int32(binary.BigEndian.Uint32(s.iv[len(s.iv)-4 : len(s.iv)]))
		s.initialized = true
	}
	s.seq++
	s.log.Printf("Seq: %d", s.seq)
	binary.BigEndian.PutUint32(s.iv[12:16], uint32(s.seq))
	s.log.Printf("IV: %v", s.iv)
	// PKCS7 padding to aes block size (16)
	neededBytes := (aes.BlockSize - (len(data))%aes.BlockSize)
	plaintext := make([]byte, len(data)+neededBytes)
	copy(plaintext, data)
	for idx := len(data); idx < len(plaintext); idx++ {
		plaintext[idx] = byte(neededBytes)
	}
	s.log.Printf("Padded plaintext: %v", plaintext)
	ciphertext, err := encryptCBC(key, s.iv[:], plaintext)
	if err != nil {
		return nil, 0, fmt.Errorf("encryption failed: %w", err)
	}
	s.log.Printf("Ciphertext: %v", ciphertext)

	// signature
	bytesToHash := append(s.getSignature(), s.iv[12:16]...)
	bytesToHash = append(bytesToHash, ciphertext...)
	s.log.Printf("Digest %d %v", len(bytesToHash), bytesToHash)
	signature := sha256.Sum256(bytesToHash)
	s.log.Printf("Signature %d %v", len(signature), signature)

	ret := append(signature[:], ciphertext...)
	s.log.Printf("Final ciphertext: %d %v", len(ret), ret)

	return ret, s.seq, nil
}

func (s *KlapSession) decrypt(data []byte) ([]byte, error) {
	plaintext, err := decryptCBC(s.key, s.iv[:], data[32:])
	if err != nil {
		return nil, err
	}
	if len(plaintext) == 0 {
		return plaintext, nil
	}
	if len(plaintext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("plaintext is not padded to AES block size")
	}
	// PKCS7 unpadding from aes block size (16)
	numPadBytes := plaintext[len(plaintext)-1]
	for n := 1; n < int(numPadBytes); n++ {
		if plaintext[len(plaintext)-n-1] != numPadBytes {
			return nil, fmt.Errorf("malformed padding")
		}
	}
	plaintext = plaintext[:len(plaintext)-int(numPadBytes)]
	s.log.Printf("Plaintext: %v", plaintext)
	return plaintext, nil
}

// AES CBC encryption, from https://gist.github.com/locked/b066aa1ddeb2b28e855e
func encryptCBC(key, iv, plaintext []byte) ([]byte, error) {
	if len(plaintext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("plaintext is not a multiple of the block size")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create new AES block cipher: %w", err)
	}

	ciphertext := make([]byte, len(plaintext))
	cbc := cipher.NewCBCEncrypter(block, iv)
	cbc.CryptBlocks(ciphertext, plaintext)

	return ciphertext, nil
}

// AES CBC decryption, from https://gist.github.com/locked/b066aa1ddeb2b28e855e
func decryptCBC(key, iv, ciphertext []byte) ([]byte, error) {
	var block cipher.Block

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create new AES block cipher: %w", err)
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	cbc := cipher.NewCBCDecrypter(block, iv)
	cbc.CryptBlocks(ciphertext, ciphertext)

	plaintext := ciphertext

	return plaintext, nil
}

func (s *KlapSession) Request(payload []byte) ([]byte, error) {
	encrypted, seq, err := s.encrypt(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt payload: %w", err)
	}
	qs := url.Values{}
	qs.Add("seq", strconv.FormatInt(int64(seq), 10))
	u := url.URL{
		Scheme:   "http",
		Host:     s.addr.String(),
		Path:     "/app/request",
		RawQuery: qs.Encode(),
	}
	s.log.Printf("Request URL: %s", u.String())
	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(encrypted))
	if err != nil {
		return nil, fmt.Errorf("http request creation failed: %w", err)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}
	c := http.Client{
		Jar: jar,
	}
	c.Jar.SetCookies(req.URL, []*http.Cookie{&http.Cookie{Name: "TP_SESSIONID", Value: s.SessionID}})
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http POST failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("expected 200 OK, got %s. Error message: %s", resp.Status, body)
	}
	decrypted, err := s.decrypt(body)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt payload: %w", err)
	}
	return decrypted, nil
}

func (s *KlapSession) Handshake(addr netip.Addr, username, password string) error {
	s.addr = addr
	if err := s.handshake1(username, password, addr); err != nil {
		return fmt.Errorf("KLAP handshake1 failed: %w", err)
	}
	return s.handshake2(addr)
}

func (s *KlapSession) handshake2(target netip.Addr) error {
	u := url.URL{
		Scheme: "http",
		Host:   target.String(),
		Path:   "/app/handshake2",
	}
	bytesToHash := append(s.RemoteSeed, s.LocalSeed...)
	bytesToHash = append(bytesToHash, s.UserHash...)
	payload := sha256.Sum256(bytesToHash)
	jar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("failed to create cookie jar: %w", err)
	}
	c := http.Client{
		Jar: jar,
	}
	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(payload[:]))
	if err != nil {
		return fmt.Errorf("http new request creation failed: %w", err)
	}
	c.Jar.SetCookies(req.URL, []*http.Cookie{&http.Cookie{Name: "TP_SESSIONID", Value: s.SessionID}})
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("http POST failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("expected 200 OK, got %s. Error message: %s", resp.Status, body)
	}
	return nil
}

func (s *KlapSession) handshake1(username, password string, target netip.Addr) error {
	u := url.URL{
		Scheme: "http",
		Host:   target.String(),
		Path:   "/app/handshake1",
	}
	var localSeed [16]byte
	if _, err := rand.Read(localSeed[:]); err != nil {
		return fmt.Errorf("failed to generate local seed: %w", err)
	}
	c := http.Client{}
	resp, err := c.Post(u.String(), "application/octet-stream", bytes.NewReader(localSeed[:]))
	if err != nil {
		return fmt.Errorf("http post failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	cookies, err := parseBrokenCookies(resp)
	if err != nil {
		return fmt.Errorf("failed to parse cookies: %w", err)
	}
	var (
		sessionID string
		expiry    time.Time
	)
	for _, c := range cookies {
		if c.Name == "TP_SESSIONID" {
			sessionID = c.Value
		} else if c.Name == "TIMEOUT" {
			timeout, err := strconv.ParseInt(c.Value, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid timeout string '%s': %w", c.Value, err)
			}
			expiry = time.Now().Add(time.Duration(timeout) * time.Second)
		}
	}
	remoteSeed := body[:16]
	serverHash := body[16:]
	var bytesToHash []byte
	calcSha1 := func(s string) []byte {
		h := sha1.Sum([]byte(s))
		return h[:]
	}
	bytesToHash = append(bytesToHash, calcSha1(username)...)
	bytesToHash = append(bytesToHash, calcSha1(password)...)
	userHash := sha256.Sum256(bytesToHash)

	bytesToHash = append(localSeed[:], remoteSeed...)
	bytesToHash = append(bytesToHash, userHash[:]...)
	localSeedAuthHash := sha256.Sum256(bytesToHash)

	if !bytes.Equal(localSeedAuthHash[:], serverHash) {
		return fmt.Errorf("authentication failed")
	}
	s.SessionID = sessionID
	s.Expiry = expiry
	s.LocalSeed = localSeed[:]
	s.RemoteSeed = remoteSeed
	s.UserHash = userHash[:]
	return nil
}

func parseBrokenCookies(r *http.Response) ([]*http.Cookie, error) {
	// Tapo's HTTP cookies are malformed, so here we go with custom parsing...
	cookieCount := len(r.Header["Set-Cookie"])
	cookies := make([]*http.Cookie, 0, cookieCount)
	if cookieCount != 0 {
		for _, line := range r.Header["Set-Cookie"] {
			parts := strings.Split(textproto.TrimString(line), ";")
			for _, part := range parts {
				name, value, ok := strings.Cut(part, "=")
				if !ok {
					continue
				}
				name = textproto.TrimString(name)
				c := &http.Cookie{
					Name:  name,
					Value: value,
					Raw:   line,
				}
				cookies = append(cookies, c)
			}
		}
	}
	return cookies, nil
}
