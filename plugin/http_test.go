package plugin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type httpHandler struct {
	val          string
	req          *http.Request
	cnt          int
	cacheBusting bool
	noDate       bool
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.req = req
	h.val = lo.RandomString(16, lo.LettersCharset)

	// emulate a device that omits the Date header (e.g. Zendure Solarflow)
	if h.noDate {
		conn, buf, err := w.(http.Hijacker).Hijack()
		if err != nil {
			panic(err)
		}
		defer conn.Close()
		// increment before flushing: the client returns as soon as it reads the body,
		// so counting after the flush races with the test asserting on cnt
		h.cnt++
		fmt.Fprintf(buf, "HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nContent-Length: %d\r\n\r\n%s", len(h.val), h.val)
		buf.Flush()
		return
	}

	if h.cacheBusting {
		w.Header().Set("Cache-Control", "no-store, no-cache, max-age=0, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
	}
	_, _ = w.Write([]byte(h.val))
	h.cnt++
}

func TestHttp(t *testing.T) {
	suite.Run(t, new(httpTestSuite))
}

type httpTestSuite struct {
	suite.Suite
	h   *httpHandler
	srv *httptest.Server
}

func (suite *httpTestSuite) SetupSuite() {
	suite.h = new(httpHandler)
	suite.srv = httptest.NewServer(suite.h)
}

func (suite *httpTestSuite) TearDown() {
	suite.srv.Close()
}

func (suite *httpTestSuite) TestGet() {
	uri := suite.srv.URL + "/foo/bar{{\"/baz\"}}"
	p := NewHTTP(util.NewLogger("foo"), http.MethodGet, uri, false, 0)

	g, err := p.StringGetter()
	suite.Require().NoError(err)

	res, err := g()
	suite.Require().NoError(err)
	suite.Require().Equal("/foo/bar/baz", suite.h.req.URL.String())
	suite.Require().Equal(suite.h.val, res)
}

func (suite *httpTestSuite) TestCacheGet() {
	uri := suite.srv.URL + "/foo/bar?baz=1"
	p := NewHTTP(util.NewLogger("foo"), http.MethodGet, uri, false, time.Minute)

	g, err := p.StringGetter()
	suite.Require().NoError(err)

	for range 3 {
		res, err := g()
		suite.Require().NoError(err)
		suite.Require().Equal("/foo/bar?baz=1", suite.h.req.URL.String())
		suite.Require().Equal(suite.h.val, res)
		suite.Require().Equal(1, suite.h.cnt)
	}
}

func (suite *httpTestSuite) TestCacheGetNoStore() {
	// upstream sends cache-busting headers, cache must still take effect (#31025)
	suite.h.cacheBusting = true
	defer func() { suite.h.cacheBusting = false }()

	uri := suite.srv.URL + "/foo/bar?baz=2"
	p := NewHTTP(util.NewLogger("foo"), http.MethodGet, uri, false, time.Minute)

	g, err := p.StringGetter()
	suite.Require().NoError(err)

	suite.h.cnt = 0
	res, err := g()
	suite.Require().NoError(err)
	first := suite.h.cnt

	for range 3 {
		val, err := g()
		suite.Require().NoError(err)
		suite.Require().Equal(res, val)
		suite.Require().Equal(first, suite.h.cnt)
	}
}

func (suite *httpTestSuite) TestCacheGetNoDate() {
	// upstream omits the Date header, cache must still take effect via injected Date
	suite.h.noDate = true
	defer func() { suite.h.noDate = false }()

	uri := suite.srv.URL + "/foo/bar?baz=3"
	p := NewHTTP(util.NewLogger("foo"), http.MethodGet, uri, false, time.Minute)

	g, err := p.StringGetter()
	suite.Require().NoError(err)

	suite.h.cnt = 0
	res, err := g()
	suite.Require().NoError(err)
	suite.Require().Equal(1, suite.h.cnt)

	for range 3 {
		val, err := g()
		suite.Require().NoError(err)
		suite.Require().Equal(res, val)
		suite.Require().Equal(1, suite.h.cnt)
	}
}

func (suite *httpTestSuite) TestSetQuery() {
	uri := suite.srv.URL + "/foo/bar?baz={{.baz}}"
	p := NewHTTP(util.NewLogger("foo"), http.MethodGet, uri, false, 0)

	s, err := p.StringSetter("baz")
	suite.Require().NoError(err)
	suite.Require().NoError(s("4711"))
	suite.Require().Equal("/foo/bar?baz=4711", suite.h.req.URL.String())
}

func (suite *httpTestSuite) TestSetPath() {
	uri := suite.srv.URL + "/foo/bar/{{.baz}}"
	p := NewHTTP(util.NewLogger("foo"), http.MethodGet, uri, false, 0)

	s, err := p.StringSetter("baz")
	suite.Require().NoError(err)
	suite.Require().NoError(s("4711"))
	suite.Require().Equal("/foo/bar/4711", suite.h.req.URL.String())
}

func TestRepeatedGet(t *testing.T) {
	url := "http://repeated.test/uncached"
	t0 := time.Now()

	require.False(t, repeatedGet(url, t0))                           // first sighting
	require.True(t, repeatedGet(url, t0.Add(500*time.Millisecond)))  // repeated within 1s: warn
	require.False(t, repeatedGet(url, t0.Add(600*time.Millisecond))) // already warned: silent

	spaced := "http://repeated.test/spaced"
	require.False(t, repeatedGet(spaced, t0))
	require.False(t, repeatedGet(spaced, t0.Add(2*time.Second))) // >1s apart: no warn

	// query params are stripped before keying, so cache-busting still counts as a repeat
	require.Equal(t, "http://q.test/path", stripQuery("http://q.test/path?ts=1&x=2#frag"))
	require.False(t, repeatedGet(stripQuery("http://q.test/path?ts=1"), t0))
	require.True(t, repeatedGet(stripQuery("http://q.test/path?ts=2"), t0.Add(300*time.Millisecond)))
}
