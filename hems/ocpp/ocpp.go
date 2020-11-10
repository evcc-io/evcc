package ocpp

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/andig/evcc/core"
	"github.com/andig/evcc/util"
	"github.com/denisbrodbeck/machineid"

	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	ocppcore "github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ws"
)

// OCPP is an OCPP client
type OCPP struct {
	log  *util.Logger
	site site
	cp   ocpp16.ChargePoint

	configuration ConfigMap
}

// site is the minimal interface for accessing site methods
type site interface {
	Configuration() core.SiteConfiguration
	LoadPoints() []core.LoadPointAPI
}

const retryTimeout = 5 * time.Second

// New generates OCPP chargepoint client
func New(conf map[string]interface{}, site site) (*OCPP, error) {
	cc := struct {
		URI       string
		StationID string
	}{}

	if err := util.DecodeOther(conf, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("ocpp")

	if cc.StationID == "" {
		id, err := machineid.ID()
		if err == nil {
			cc.StationID = fmt.Sprintf("evcc-%s", strings.ToLower(id))
		} else {
			cc.StationID = fmt.Sprintf("evcc-%d", rand.Int31())
		}
		log.DEBUG.Println("station id:", cc.StationID)
	}

	ws := ws.NewClient()
	cp := ocpp16.NewChargePoint(cc.StationID, nil, ws)

	s := &OCPP{
		log:           log,
		site:          site,
		cp:            cp,
		configuration: getDefaultConfig(),
	}

	err := cp.Start(cc.URI)
	if err == nil {
		cp.SetCoreHandler(s)
		go s.errorHandler(ws.Errors())
		go s.errorHandler(cp.Errors())
	}

	return s, err
}

// errorHandler logs error channel
func (s *OCPP) errorHandler(errC <-chan error) {
	for err := range errC {
		s.log.ERROR.Println(err)
	}
}

// Run executes the OCPP chargepoint client
func (s *OCPP) Run() {
	for {
		for id, lp := range s.site.LoadPoints() {
			connector := id + 1

			status := ocppcore.ChargePointStatusAvailable
			if lp.GetCharging() {
				status = ocppcore.ChargePointStatusCharging
			}

			s.log.TRACE.Printf("send: %s status: %+v", lp.Name(), status)
			if _, err := s.cp.StatusNotification(connector, ocppcore.NoError, status); err != nil {
				s.log.ERROR.Printf("%s: %v", lp.Name(), err)
			}
		}

		time.Sleep(retryTimeout)
	}
}
