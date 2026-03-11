package warp

import (
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jpfielding/go-http-digest/pkg/digest"
)

type Connection struct {
	*request.Helper
	URI      string
	Username string
	Password string
}

func NewConnection(log *util.Logger, uri, user, pass string) *Connection {
	c := &Connection{
		Helper:   request.NewHelper(log),
		URI:      util.DefaultScheme(strings.TrimRight(uri, "/"), "http"),
		Username: user,
		Password: pass,
	}

	if c.Username != "" && c.Password != "" {
		c.Client.Transport = digest.NewTransport(c.Username, c.Password, c.Client.Transport)
	}

	return c
}
