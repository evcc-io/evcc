package vaillant

import (
	"bytes"
	"context"
	"crypto/pbkdf2"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/WulfgarW/sensonet"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

const altchaChallengeURL = "https://identity.vaillant-group.com/api/altcha/challenge"

// Login replicates sensonet.Oauth2Config.PasswordCredentialsToken with the
// ALTCHA proof-of-work the Vaillant login requires (https://github.com/signalkraft/myPyllant/pull/162)
func Login(ctx context.Context, log *util.Logger, oc *sensonet.Oauth2Config, username, password string) (*oauth2.Token, error) {
	client := new(http.Client)
	if c, ok := ctx.Value(oauth2.HTTPClient).(*http.Client); ok {
		// shallow copy to avoid mutating the shared client
		clone := *c
		client = &clone
	}

	client.Jar, _ = cookiejar.New(nil)
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	cv := oauth2.GenerateVerifier()

	uri := oc.AuthCodeURL(cv, oauth2.S256ChallengeOption(cv), oauth2.SetAuthURLParam("code", "code_challenge"))
	resp, err := client.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	match := regexp.MustCompile(`action\s*=\s*"(.+?)"`).FindStringSubmatch(string(body))
	if len(match) < 2 {
		return nil, errors.New("missing login form action")
	}

	params := url.Values{
		"username":     {username},
		"password":     {password},
		"credentialId": {""},
	}

	// best-effort like myPyllant: continue without altcha if challenge cannot be obtained
	if altcha, err := altcha(client); err == nil {
		params.Set("altcha", altcha)
	} else {
		log.WARN.Printf("altcha challenge failed, continuing without: %v", err)
	}

	req, err := http.NewRequest("POST", match[1], strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	location, _ := url.Parse(resp.Header.Get("Location"))
	code := location.Query().Get("code")
	if code == "" {
		return nil, errors.New("could not get code")
	}

	return oc.Exchange(ctx, code, oauth2.VerifierOption(cv))
}

// altcha fetches and solves the ALTCHA challenge for the login form
func altcha(client *http.Client) (string, error) {
	resp, err := client.Get(altchaChallengeURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %s", resp.Status)
	}

	challenge, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return solveAltcha(challenge)
}

// solveAltcha solves the PBKDF2 proof-of-work and returns the base64-encoded
// payload the login form expects in its altcha field
func solveAltcha(challenge []byte) (string, error) {
	var c struct {
		Parameters json.RawMessage `json:"parameters"`
		Signature  string          `json:"signature"`
	}
	if err := json.Unmarshal(challenge, &c); err != nil {
		return "", err
	}

	var p struct {
		Algorithm string `json:"algorithm"`
		Cost      int    `json:"cost"`
		KeyLength int    `json:"keyLength"`
		KeyPrefix string `json:"keyPrefix"`
		Nonce     string `json:"nonce"`
		Salt      string `json:"salt"`
	}
	if err := json.Unmarshal(c.Parameters, &p); err != nil {
		return "", err
	}

	nonce, err := hex.DecodeString(p.Nonce)
	if err != nil {
		return "", err
	}
	salt, err := hex.DecodeString(p.Salt)
	if err != nil {
		return "", err
	}
	prefix, err := hex.DecodeString(p.KeyPrefix)
	if err != nil {
		return "", err
	}

	newHash := sha256.New
	switch p.Algorithm {
	case "PBKDF2/SHA-512":
		newHash = sha512.New
	case "PBKDF2/SHA-384":
		newHash = sha512.New384
	}

	keyLength := p.KeyLength
	if keyLength == 0 {
		keyLength = 32
	}

	for counter := uint32(0); ; counter++ {
		password := binary.BigEndian.AppendUint32(slices.Clone(nonce), counter)

		key, err := pbkdf2.Key(newHash, string(password), salt, p.Cost, keyLength)
		if err != nil {
			return "", err
		}

		if !bytes.HasPrefix(key, prefix) {
			continue
		}

		payload := struct {
			Challenge struct {
				Parameters json.RawMessage `json:"parameters"`
				Signature  string          `json:"signature"`
			} `json:"challenge"`
			Solution struct {
				Counter    uint32 `json:"counter"`
				DerivedKey string `json:"derivedKey"`
				Time       int    `json:"time"`
			} `json:"solution"`
		}{}
		payload.Challenge.Parameters = c.Parameters
		payload.Challenge.Signature = c.Signature
		payload.Solution.Counter = counter
		payload.Solution.DerivedKey = hex.EncodeToString(key)

		res, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}

		return base64.StdEncoding.EncodeToString(res), nil
	}
}
