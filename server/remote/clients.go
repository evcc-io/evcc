package remote

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/samber/lo"
	"golang.org/x/crypto/bcrypt"
)

// dummyHash is a bcrypt hash of a random value, used to make the
// "unknown user" path take the same time as a real password check and
// prevent username enumeration via timing side channels.
var dummyHash []byte

func init() {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	h, err := bcrypt.GenerateFromPassword(buf, bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	dummyHash = h
}

// Client is a single tunnel basic-auth credential used by a remote client.
type Client struct {
	Username  string     `json:"username"`
	Hash      string     `json:"-"`
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// clientsEnvelope wraps the persisted list so Hash can be stored too.
type clientsEnvelope struct {
	Clients []persistedClient `json:"clients"`
}

type persistedClient struct {
	Username  string     `json:"username"`
	Hash      string     `json:"hash"`
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// loadClients reads the persisted client list.
func loadClients() []persistedClient {
	var res clientsEnvelope
	_ = settings.Json(keys.RemoteClients, &res)
	return res.Clients
}

// saveClients persists the given client list.
func saveClients(list []persistedClient) error {
	return settings.SetJson(keys.RemoteClients, clientsEnvelope{Clients: list})
}

// generatePassword returns a crypto-random uppercase base32 password formatted
// as two groups of eight characters separated by a hyphen, e.g. "NLDJ6FYF-LADB5UZH".
// 16 base32 chars = 80 bits of entropy.
func generatePassword() (string, error) {
	buf := make([]byte, 10) // 10 bytes -> 16 base32 chars
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	s := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
	return s[0:8] + "-" + s[8:16], nil
}

// Clients returns the list of configured clients (without password hashes).
func (r *Remote) Clients() []Client {
	list := loadClients()
	out := make([]Client, 0, len(list))
	for _, c := range list {
		out = append(out, Client{
			Username:  c.Username,
			CreatedAt: c.CreatedAt,
			ExpiresAt: c.ExpiresAt,
		})
	}
	return out
}

// CreateClient creates a new client with an auto-generated password.
// expiresIn <= 0 means the client never expires.
// Returns the cleartext password (shown to the user only once).
func (r *Remote) CreateClient(username string, expiresIn time.Duration) (Client, string, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return Client{}, "", errors.New("username required")
	}
	// RFC 7617: ":" is the basic-auth separator; reject control chars too.
	for _, r := range username {
		if r == ':' || r < 0x20 || r == 0x7f {
			return Client{}, "", errors.New("username contains invalid characters")
		}
	}

	var expires *time.Time
	if expiresIn > 0 {
		expires = new(time.Now().Add(expiresIn))
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	list := loadClients()
	for _, c := range list {
		if c.Username == username {
			return Client{}, "", fmt.Errorf("client %q already exists", username)
		}
	}

	password, err := generatePassword()
	if err != nil {
		return Client{}, "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Client{}, "", err
	}

	c := persistedClient{
		Username:  username,
		Hash:      string(hash),
		CreatedAt: time.Now(),
		ExpiresAt: expires,
	}
	list = append(list, c)
	if err := saveClients(list); err != nil {
		return Client{}, "", err
	}

	return Client{
		Username:  c.Username,
		CreatedAt: c.CreatedAt,
		ExpiresAt: c.ExpiresAt,
	}, password, nil
}

// DeleteClient removes a client by username.
func (r *Remote) DeleteClient(username string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	list := loadClients()
	out := lo.Reject(list, func(c persistedClient, _ int) bool {
		return c.Username == username
	})
	if len(out) == len(list) {
		return fmt.Errorf("client %q not found", username)
	}
	return saveClients(out)
}

// Authenticate validates basic-auth credentials. Always runs bcrypt
// (against a dummy hash on miss) to prevent username enumeration via timing.
func (r *Remote) Authenticate(username, password string) bool {
	hash := dummyHash
	var found *persistedClient
	for _, c := range loadClients() {
		if c.Username == username {
			found = &c
			hash = []byte(c.Hash)
			break
		}
	}

	valid := bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil
	if !valid || found == nil {
		return false
	}
	if found.ExpiresAt != nil && time.Now().After(*found.ExpiresAt) {
		return false
	}
	return true
}
