package ocpp

import (
	"time"

	"github.com/andig/evcc/core"
	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"

	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	ocppcore "github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
)

// OCPP is an OCPP client
type OCPP struct {
	log   *util.Logger
	cache *util.Cache
	site  site
	uri   string
	cp    ocpp16.ChargePoint

	connectors    map[int]*ConnectorInfo
	configuration ConfigMap
}

type ConnectorInfo struct {
	status             ocppcore.ChargePointStatus
	availability       ocppcore.AvailabilityType
	currentTransaction int
	currentReservation int
}

// site is the minimal interface for accessing site methods
type site interface {
	Configuration() core.SiteConfiguration
	LoadPoints() []core.LoadPointAPI
}

const retryTimeout = 30 * time.Second

// New generates OCPP chargepoint client
func New(conf map[string]interface{}, site site, cache *util.Cache, httpd *server.HTTPd) (*OCPP, error) {
	cc := struct {
		URI       string
		StationID string
	}{
		StationID: "evcc",
	}

	if err := util.DecodeOther(conf, &cc); err != nil {
		return nil, err
	}

	cp := ocpp16.NewChargePoint(cc.StationID, nil, nil)

	s := &OCPP{
		log:   util.NewLogger("ocpp"),
		cache: cache,
		site:  site,
		uri:   cc.URI,
		cp:    cp,
	}

	cp.SetCoreHandler(s)

	return s, nil
}

// Run executes the OCPP chargepoint client
func (s *OCPP) Run() {
	for {
		if err := s.cp.Start(s.uri); err != nil {
			s.log.ERROR.Println(err)
		}

		time.Sleep(retryTimeout)
	}
}
