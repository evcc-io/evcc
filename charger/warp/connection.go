package warp

import (
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/icholy/digest"
)

func NewConnection(log *util.Logger, uri, user, pass string) *Connection {
	c := &Connection{
		Helper:   request.NewHelper(log),
		URI:      util.DefaultScheme(strings.TrimRight(uri, "/"), "http"),
		Username: user,
		Password: pass,
	}
	c.ApplyDigest()
	return c
}

func (c *Connection) ApplyDigest() {
	if c.Username != "" && c.Password != "" {
		c.Client.Transport = &digest.Transport{
			Username:  c.Username,
			Password:  c.Password,
			Transport: c.Client.Transport,
		}
	}
}
