package provider

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

type httpHandler struct {
	val string
	req *http.Request
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.req = req
	h.val = lo.RandomString(16, lo.LettersCharset)
	_, _ = w.Write([]byte(h.val))
}

func TestHttpGet(t *testing.T) {
	h := new(httpHandler)
	srv := httptest.NewServer(h)
	defer srv.Close()

	uri := srv.URL + "/foo/bar"
	p := NewHTTP(util.NewLogger("foo"), http.MethodGet, uri, false, 1, 0)

	uriUrl, _ := url.Parse(uri)

	res, err := p.StringGetter()()
	assert.NoError(t, err)
	assert.Equal(t, uriUrl.Path, h.req.URL.Path)
	assert.Equal(t, h.val, res)
}

func TestHttpSet(t *testing.T) {
	h := new(httpHandler)
	srv := httptest.NewServer(h)
	defer srv.Close()

	uri := srv.URL + "/foo/bar?baz={{.baz}}"
	p := NewHTTP(util.NewLogger("foo"), http.MethodGet, uri, false, 1, 0)

	uriUrl, _ := url.Parse(uri)

	err := p.StringSetter("baz")("4711")
	assert.NoError(t, err)
	assert.Equal(t, uriUrl.Path, h.req.URL.Path)
	assert.Equal(t, "baz=4711", h.req.URL.RawQuery)
}
