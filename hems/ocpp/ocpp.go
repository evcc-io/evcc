package ocpp

import (
	"context"
	"fmt"
	"time"

	"github.com/andig/evcc/core"
	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"

	gocpp "github.com/eduhenke/go-ocpp"
	"github.com/eduhenke/go-ocpp/cp"
	"github.com/eduhenke/go-ocpp/messages/v1x/cpreq"
	"github.com/eduhenke/go-ocpp/messages/v1x/cpresp"
	"github.com/eduhenke/go-ocpp/messages/v1x/csreq"
	"github.com/eduhenke/go-ocpp/messages/v1x/csresp"
)

// OCPP is an OCPP client
type OCPP struct {
	log    *util.Logger
	cache  *util.Cache
	closeC chan struct{}
	doneC  chan struct{}
	uri    string
	site   site
	client cp.ChargePoint
}

// site is the minimal interface for accessing site methods
type site interface {
	Configuration() core.SiteConfiguration
	LoadPoints() []core.LoadPointAPI
}

// New generates SEMP Gateway listening at /semp endpoint
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

	client, err := cp.New(cc.StationID, cc.URI, gocpp.V16, gocpp.JSON) // or ocpp.SOAP
	if err != nil {
		return nil, fmt.Errorf("could not create ocpp client: %v", err)
	}

	s := &OCPP{
		doneC:  make(chan struct{}),
		log:    util.NewLogger("ocpp"),
		cache:  cache,
		site:   site,
		client: client,
	}

	if err := s.boot(); err != nil {
		return nil, fmt.Errorf("could not connect to ocpp central system: %v", err)
	}

	return s, nil
}

// Run executes the SEMP runtime
func (s *OCPP) Run() {
	go s.heartbeat()

	s.client.Run(context.Background(), nil, s.handler)
}

func (s *OCPP) handler(req csreq.CentralSystemRequest) (csresp.CentralSystemResponse, error) {
	var resp csresp.CentralSystemResponse
	s.log.TRACE.Printf("recv: %+v", req)

	switch req := req.(type) {
	case *csreq.SetChargingProfile:
		resp = &csresp.SetChargingProfile{
			Status: "ok",
		}
	default:
		return nil, fmt.Errorf("invalid request: %v", req)
	}

	return resp, nil
}

func (s *OCPP) boot() error {
	req := &cpreq.BootNotification{
		ChargePointModel:  "evcc",
		ChargePointVendor: "github.com/andig/evcc",
	}

	raw, err := s.client.Send(req)
	if err == nil {
		resp, ok := raw.(*cpresp.BootNotification)
		s.log.TRACE.Printf("recv: %+v", resp)

		if !ok {
			err = fmt.Errorf("invalid boot response: %+v", err)
		}
	}

	return err
}

func (s *OCPP) heartbeat() {
	for {
		time.Sleep(5 * time.Second)

		raw, err := s.client.Send(&cpreq.Heartbeat{})
		if err != nil {
			s.log.ERROR.Printf("send failed: %+v", err)
			continue
		}

		_, ok := raw.(*cpresp.Heartbeat)
		if !ok {
			s.log.ERROR.Printf("invalid heartbeat response: %+v", err)
		}
	}
}
