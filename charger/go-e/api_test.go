package goe

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/evcc-io/evcc/util"
)

type handler struct {
	uri string
}

func (h *handler) expect(uri string) {
	h.uri = uri
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if r.URL.RawQuery != "" {
		path += "?" + r.URL.RawQuery
	}
	fmt.Println(path)

	if path == h.uri {
		fmt.Fprint(w, "{}")
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "expected %s", h.uri)
	}
}

func TestLocalV1(t *testing.T) {
	h := &handler{}
	srv := httptest.NewServer(h)

	// h.expect("/api/status?filter=alw")
	local := NewLocal(util.NewLogger("foo"), srv.URL, 0)

	h.expect("/status")
	if _, err := local.Status(); err != nil {
		t.Error(err)
	}

	h.expect("/mqtt?payload=foo=bar")
	if err := local.Update("foo=bar"); err != nil {
		t.Error(err)
	}
}

// TestLocalConcurrentStatus exercises the status cache from many goroutines at
// once (as the fast sense loop and the control loop do). With cache=0 every call
// refreshes, so without the mutex the shared status/updated fields race; run with
// -race to verify the cache is concurrency-safe.
func TestLocalConcurrentStatus(t *testing.T) {
	h := &handler{}
	srv := httptest.NewServer(h)
	defer srv.Close()

	local := NewLocal(util.NewLogger("foo"), srv.URL, 0) // upgradeV2 fails -> v1
	h.expect("/status")

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := local.Status(); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

func TestLocalV2(t *testing.T) {
	h := &handler{}
	srv := httptest.NewServer(h)

	h.expect("/api/status?filter=alw")
	local := NewLocal(util.NewLogger("foo"), srv.URL, 0)

	h.expect("/api/status?filter=alw,car,eto,nrg,wh,trx,cards")
	if _, err := local.Status(); err != nil {
		t.Error(err)
	}

	h.expect("/api/set?foo=bar")
	if err := local.Update("foo=bar"); err != nil {
		t.Error(err)
	}
}
