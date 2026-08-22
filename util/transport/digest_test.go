package transport

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	digestUser  = "admin"
	digestPass  = "secret"
	digestRealm = "shellypro4pm-f008d1d8b8b8"
)

var reDigestParam = regexp.MustCompile(`(\w+)=(?:"([^"]*)"|([^,\s]+))`)

func digestParams(auth string) map[string]string {
	res := make(map[string]string)
	for _, m := range reDigestParam.FindAllStringSubmatch(auth, -1) {
		if m[2] != "" {
			res[m[1]] = m[2]
		} else {
			res[m[1]] = m[3]
		}
	}
	return res
}

func sha256hex(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, ":")))
	return hex.EncodeToString(sum[:])
}

// digestDevice verifies digest credentials the way a Shelly Gen2+ device does:
// one nonce per challenge, reusable while nc is strictly increasing.
type digestDevice struct {
	mu sync.Mutex

	algorithm string // algorithm advertised in the challenge
	nonce     string
	seq       int
	lastNC    int

	challenges int // nonces minted
	unauth     int // requests without Authorization
	ncSeen     []int
	seenAlg    string // algorithm echoed by the client
}

func (d *digestDevice) challengeLocked(w http.ResponseWriter) {
	d.seq++
	d.nonce = fmt.Sprintf("nonce-%d", d.seq)
	d.lastNC = 0
	d.challenges++
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Digest qop="auth", realm=%q, nonce=%q, algorithm=%s`,
		digestRealm, d.nonce, d.algorithm))
	w.WriteHeader(http.StatusUnauthorized)
}

func (d *digestDevice) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.mu.Lock()
	defer d.mu.Unlock()

	auth := r.Header.Get("Authorization")
	if auth == "" {
		d.unauth++
		d.challengeLocked(w)
		return
	}

	p := digestParams(auth)
	d.seenAlg = p["algorithm"]
	nc, err := strconv.ParseInt(p["nc"], 16, 64)
	if p["nonce"] != d.nonce || err != nil || int(nc) <= d.lastNC {
		d.challengeLocked(w)
		return
	}

	// the device always hashes with SHA-256, whatever spelling it advertised
	ha1 := sha256hex(digestUser, digestRealm, digestPass)
	ha2 := sha256hex(r.Method, r.URL.RequestURI())
	if want := sha256hex(ha1, d.nonce, p["nc"], p["cnonce"], "auth", ha2); want != p["response"] {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	d.lastNC = int(nc)
	d.ncSeen = append(d.ncSeen, int(nc))
	w.WriteHeader(http.StatusOK)
}

func testDigestClient(t *testing.T, algorithm string) (*digestDevice, *http.Client, string) {
	t.Helper()

	dev := &digestDevice{algorithm: algorithm}
	srv := httptest.NewServer(dev)
	t.Cleanup(srv.Close)

	return dev, &http.Client{Transport: Digest(digestUser, digestPass, nil)}, srv.URL
}

// TestDigestPreemptive asserts the challenge is reused. Devices bound the
// number of nonces they issue, so re-challenging per request gets us throttled.
func TestDigestPreemptive(t *testing.T) {
	dev, client, uri := testDigestClient(t, "SHA-256")

	for i := 0; i < 4; i++ {
		resp, err := client.Get(uri + "/rpc/Switch.GetStatus")
		require.NoError(t, err, "request %d", i)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode, "request %d", i)
	}

	dev.mu.Lock()
	defer dev.mu.Unlock()
	assert.Equal(t, 1, dev.challenges, "only the first request may trigger a challenge")
	assert.Equal(t, 1, dev.unauth, "only the first request may be unauthenticated")
	assert.Equal(t, []int{1, 2, 3, 4}, dev.ncSeen, "nc must increase across reuse")
}

// TestDigestNonRfcAlgorithm covers servers advertising SHA256 instead of the
// RFC 7616 compliant SHA-256. The non-RFC spelling takes a separate challenge
// path, so it has to reuse challenges just the same.
func TestDigestNonRfcAlgorithm(t *testing.T) {
	for _, algorithm := range []string{"SHA-256", "SHA256"} {
		t.Run(algorithm, func(t *testing.T) {
			dev, client, uri := testDigestClient(t, algorithm)

			for i := 0; i < 3; i++ {
				resp, err := client.Get(uri)
				require.NoError(t, err, "request %d", i)
				resp.Body.Close()
				require.Equal(t, http.StatusOK, resp.StatusCode, "request %d", i)
			}

			dev.mu.Lock()
			defer dev.mu.Unlock()
			assert.Equal(t, algorithm, dev.seenAlg, "client must echo the announced algorithm")
			assert.Equal(t, 1, dev.challenges)
			assert.Equal(t, 1, dev.unauth)
			assert.Equal(t, []int{1, 2, 3}, dev.ncSeen)
		})
	}
}

// TestDigestWrongPassword asserts a rejected response surfaces to the caller
// instead of being retried until the device throttles us.
func TestDigestWrongPassword(t *testing.T) {
	dev, _, uri := testDigestClient(t, "SHA-256")
	client := &http.Client{Transport: Digest(digestUser, "wrong", nil)}

	resp, err := client.Get(uri)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	dev.mu.Lock()
	defer dev.mu.Unlock()
	assert.Equal(t, 1, dev.challenges, "a rejected response must not be retried")
	assert.Equal(t, 1, dev.unauth)
	assert.Empty(t, dev.ncSeen)
}
