package warp

import (
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

func NewConnection(log *util.Logger, uri, user, pass string) *Connection {
	c := &Connection{
		Helper:   request.NewHelper(log, user, pass),
		URI:      util.DefaultScheme(strings.TrimRight(uri, "/"), "http"),
	}
	return c
}
