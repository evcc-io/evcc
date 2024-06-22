package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
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
	p := NewHTTP(util.NewLogger("foo"), http.MethodGet, uri, false, 1, 0)

	g, err := p.StringGetter()
	suite.Require().NoError(err)

	res, err := g()
	suite.Require().NoError(err)
	suite.Require().Equal("/foo/bar/baz", suite.h.req.URL.String())
	suite.Require().Equal(suite.h.val, res)
}

func (suite *httpTestSuite) TestSetQuery() {
	uri := suite.srv.URL + "/foo/bar?baz={{.baz}}"
	p := NewHTTP(util.NewLogger("foo"), http.MethodGet, uri, false, 1, 0)

	s, err := p.StringSetter("baz")
	suite.Require().NoError(err)
	suite.Require().NoError(s("4711"))
	suite.Require().Equal("/foo/bar?baz=4711", suite.h.req.URL.String())
}

func (suite *httpTestSuite) TestSetPath() {
	uri := suite.srv.URL + "/foo/bar/{{.baz}}"
	p := NewHTTP(util.NewLogger("foo"), http.MethodGet, uri, false, 1, 0)

	s, err := p.StringSetter("baz")
	suite.Require().NoError(err)
	suite.Require().NoError(s("4711"))
	suite.Require().Equal("/foo/bar/4711", suite.h.req.URL.String())
}
