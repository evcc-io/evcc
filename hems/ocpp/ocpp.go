package ocpp

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/ocpp/profile"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/logx"

	"github.com/denisbrodbeck/machineid"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	ocppcore "github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ws"
)

// OCPP is an OCPP client
type OCPP struct {
	log  logx.Logger
	site site.API
	cp   ocpp16.ChargePoint
}

const retryTimeout = 5 * time.Second

// New generates OCPP chargepoint client
func New(conf map[string]interface{}, site site.API) (*OCPP, error) {
	cc := struct {
		URI       string
		StationID string
	}{}

	if err := util.DecodeOther(conf, &cc); err != nil {
		return nil, err
	}

	log := logx.NewModule("ocpp")

	if cc.StationID == "" {
		id, err := machineid.ProtectedID("evcc-ocpp")
		if err == nil {
			cc.StationID = fmt.Sprintf("evcc-%s", strings.ToLower(id))
		} else {
			cc.StationID = fmt.Sprintf("evcc-%d", rand.Int31())
		}
		logx.Debug(log, "station id", cc.StationID)
	}

	ws := ws.NewClient()
	cp := ocpp16.NewChargePoint(cc.StationID, nil, ws)

	s := &OCPP{
		log:  log,
		site: site,
		cp:   cp,
	}

	err := cp.Start(cc.URI)
	if err == nil {
		cp.SetCoreHandler(profile.NewCore(log, profile.GetDefaultConfig()))
		cp.SetSmartChargingHandler(profile.NewSmartCharging(log))

		go s.errorHandler(ws.Errors())
		go s.errorHandler(cp.Errors())
	}

	return s, err
}

// errorHandler logs error channel
func (s *OCPP) errorHandler(errC <-chan error) {
	for err := range errC {
		logx.Error(s.log, "error", err)
	}
}

// Run executes the OCPP chargepoint client
func (s *OCPP) Run() {
	for {
		for id, lp := range s.site.LoadPoints() {
			connector := id + 1

			status := ocppcore.ChargePointStatusAvailable
			if lp.GetStatus() == api.StatusC {
				status = ocppcore.ChargePointStatusCharging
			}

			logx.Debug(s.log, "lp", connector, "status", status)
			if _, err := s.cp.StatusNotification(connector, ocppcore.NoError, status); err != nil {
				logx.Error(s.log, "lp", connector, "error", err)
			}
		}

		time.Sleep(retryTimeout)
	}
}
