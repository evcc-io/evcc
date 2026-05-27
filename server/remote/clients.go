package remote

import (
	"crypto/rand"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/samber/lo"
	"github.com/sethvargo/go-password/password"
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
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

type persistedClient struct {
	Client
	Hash string `json:"hash"`
}

// loadClients reads the persisted client list.
func loadClients() []persistedClient {
	var res []persistedClient
	_ = settings.Json(keys.RemoteClients, &res)
	return res
}

// saveClients persists the given client list.
func saveClients(list []persistedClient) error {
	return settings.SetJson(keys.RemoteClients, list)
}

// generatePassword returns a crypto-random alphanumeric password
// with 20 characters including 4 digits (~96 bits of entropy).
func generatePassword() (string, error) {
	return password.Generate(20, 4, 0, false, false)
}

// Clients returns the list of configured clients (without password hashes).
func (r *Remote) Clients() []Client {
	return lo.Map(loadClients(), func(c persistedClient, _ int) Client {
		return c.Client
	})
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
		Client: Client{
			Username:  username,
			CreatedAt: time.Now(),
			ExpiresAt: expires,
		},
		Hash: string(hash),
	}
	list = append(list, c)
	if err := saveClients(list); err != nil {
		return Client{}, "", err
	}

	return c.Client, password, nil
}

// DeleteClient removes a client by username.
func (r *Remote) DeleteClient(username string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	list := loadClients()
	idx := slices.IndexFunc(list, func(c persistedClient) bool {
		return c.Username == username
	})

	if idx == -1 {
		return fmt.Errorf("client %s not found", username)
	}

	return saveClients(slices.Delete(list, idx, idx+1))
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
