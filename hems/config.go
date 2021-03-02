package hems

import (
	"errors"
	"strings"

	"github.com/mark-sch/evcc/core"
	"github.com/mark-sch/evcc/hems/ocpp"
	"github.com/mark-sch/evcc/hems/semp"
	"github.com/mark-sch/evcc/server"
	"github.com/mark-sch/evcc/util"
)

// HEMS describes the HEMS system interface
type HEMS interface {
	Run()
}

// NewFromConfig creates new HEMS from config
func NewFromConfig(typ string, other map[string]interface{}, site *core.Site, cache *util.Cache, httpd *server.HTTPd) (HEMS, error) {
	switch strings.ToLower(typ) {
	case "sma", "shm", "semp":
		return semp.New(other, site, cache, httpd)
	case "ocpp":
		return ocpp.New(other, site, cache)
	default:
		return nil, errors.New("unknown hems: " + typ)
	}
}
