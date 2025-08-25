package hems

import (
	"context"
	"errors"
	"strings"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/eebus"
	"github.com/evcc-io/evcc/hems/relay"
	"github.com/evcc-io/evcc/hems/shm"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util"
)

// HEMS describes the HEMS system interface
type HEMS interface {
	Run()
}

// NewFromConfig creates new HEMS from config
func NewFromConfig(ctx context.Context, typ string, other map[string]interface{}, site site.API, httpd *server.HTTPd) (HEMS, error) {
	switch strings.ToLower(typ) {
	case "sma", "shm", "semp":
		util.NewLogger("main").WARN.Println("configuring SMA Sunny Home Manager as HEMS is deprecated and will be removed in a future version")
		return shm.NewFromConfig(other, site, httpd.Addr, httpd.Router())
	case "eebus":
		return eebus.NewFromConfig(ctx, other, site)
	case "relay":
		return relay.NewFromConfig(ctx, other, site)
	default:
		return nil, errors.New("unknown hems: " + typ)
	}
}
