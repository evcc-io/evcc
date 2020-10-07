package hems

import (
	"errors"
	"strings"

	"github.com/mark-sch/evcc/core"
	"github.com/mark-sch/evcc/hems/semp"
	"github.com/mark-sch/evcc/server"
	"github.com/mark-sch/evcc/util"
)

// HEMS describes the HEMS system interface
type HEMS interface {
	Run()
}

// NewFromConfig creates new HEMS from config
func NewFromConfig(typ string, site *core.Site, cache *util.Cache, httpd *server.HTTPd) (HEMS, error) {
	switch strings.ToLower(typ) {
	case "sma", "shm", "semp":
		return semp.New(site, cache, httpd)
	default:
		return nil, errors.New("unknown hems: " + typ)
	}
}
