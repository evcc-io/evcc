package main

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/ocpp/profile"
	"github.com/evcc-io/evcc/util"

	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	ocppcore "github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ws"
)

// OCPP is an OCPP client
type OCPP struct {
	log   *util.Logger
	cache *util.Cache
	site  site.API
	cp    ocpp16.ChargePoint
}

const retryTimeout = 5 * time.Second

// New generates OCPP chargepoint client
func New(conf map[string]interface{}) (*OCPP, error) {
	cc := struct {
		URI       string
		StationID string
	}{
		StationID: "horst",
	}

	if err := util.DecodeOther(conf, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("ocpp")

	ws := ws.NewClient()
	cp := ocpp16.NewChargePoint(cc.StationID, nil, ws)

	s := &OCPP{
		log: log,
		cp:  cp,
	}

	fmt.Println("ws connect:", cc.URI)

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
		s.log.ERROR.Println(err)
	}
}

// Run executes the OCPP chargepoint client
func (s *OCPP) Run() {
	connector := 0

	status := ocppcore.ChargePointStatusAvailable

	s.log.DEBUG.Printf("send: lp-%d status: %+v", connector, status)
	if _, err := s.cp.StatusNotification(connector, ocppcore.NoError, status); err != nil {
		s.log.ERROR.Printf("lp-%d: %v", connector, err)
	}
}

func main() {
	cp, err := New(map[string]interface{}{
		"uri":       "ws://localhost:8887",
		"stationid": "horst",
	})
	if err != nil {
		panic(err)
	}

	for range time.NewTicker(1 * time.Second).C {
		fmt.Println("send status")
		cp.Run()
	}
}
