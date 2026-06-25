package plugin

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
)

type httpHandler struct {
	val          string
	req          *http.Request
	cnt          int
	cacheBusting bool
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.req = req
	h.val = lo.RandomString(16, lo.LettersCharset)
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
