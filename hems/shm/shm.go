package shm

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/machine"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/koron/go-ssdp"
)

const (
	sempController   = "Sunny Home Manager"
	sempBaseURLEnv   = "SEMP_BASE_URL"
	sempGateway      = "urn:schemas-simple-energy-management-protocol:device:Gateway:1"
	sempDeviceId     = "F-%s-%.12x-00" // 6 bytes
	sempSerialNumber = "%s-%d"
	sempCharger      = "EVCharger"
	basePath         = "/semp"
	maxAge           = 1800
)

var serverName = "EVCC SEMP Server " + util.Version

// SEMP is the SMA SEMP server
type SEMP struct {
	log     *util.Logger
	vid     string
	did     []byte
	uid     string
	hostURI string
	port    int
	site    site.API
}

type Config struct {
	AllowControl_ bool   `json:"allowControl,omitempty"` // deprecated
	VendorId      string `json:"vendorId"`
	DeviceId      string `json:"deviceId"`
}

// NewFromConfig creates a new SEMP instance from configuration and starts it
func NewFromConfig(cfg Config, site site.API, addr string, router *mux.Router) error {
	vendorId := cfg.VendorId
	if vendorId == "" {
		vendorId = "28081973"
	} else if len(vendorId) != 8 {
		return fmt.Errorf("invalid vendor id: %v. Must be 8 characters HEX string", vendorId)
	}

	uid, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	var did []byte
	if cfg.DeviceId == "" {
		if did, err = UniqueDeviceID(); err != nil {
			return fmt.Errorf("creating device id: %w", err)
		}
	} else {
		if did, err = hex.DecodeString(cfg.DeviceId); err != nil {
			return fmt.Errorf("device id: %w", err)
		}
	}

	if len(did) != 6 {
		return fmt.Errorf("invalid device id: %v. Must be 12 characters HEX string", cfg.DeviceId)
	}

	s := &SEMP{
		log:  util.NewLogger("semp"),
		site: site,
		uid:  uid.String(),
		vid:  vendorId,
		did:  did,
	}

	// find external port
	// TODO refactor network config
	_, port, err := net.SplitHostPort(addr)
	if err == nil {
		s.port, err = strconv.Atoi(port)
	}
	if err != nil {
		return err
	}

	s.hostURI = s.callbackURI()

	s.handlers(router)

	go s.run()
	return nil
}

func (s *SEMP) advertise(st, usn string) (*ssdp.Advertiser, error) {
	descriptor := s.hostURI + basePath + "/description.xml"
	return ssdp.Advertise(st, usn, descriptor, serverName, maxAge)
}

// run executes the SEMP runtime
func (s *SEMP) run() {
	uid := "uuid:" + s.uid

	var ads []*ssdp.Advertiser
	for _, ad := range []struct{ st, usn string }{
		{ssdp.RootDevice, uid + "::" + ssdp.RootDevice},
		{sempGateway, uid + "::" + sempGateway},
		{uid, uid},
	} {
		ad, err := s.advertise(ad.st, ad.usn)
		if err != nil {
			s.log.ERROR.Printf("advertise: %v", err)
			continue
		}
		ads = append(ads, ad)
	}

	for range time.Tick(maxAge * time.Second / 2) {
		for _, ad := range ads {
			if err := ad.Alive(); err != nil {
				s.log.ERROR.Println(err)
			}
		}
	}
}

func (s *SEMP) callbackURI() string {
	if uri := os.Getenv(sempBaseURLEnv); uri != "" {
		return strings.TrimSuffix(uri, "/")
	}

	ip := "localhost"
	ips := util.LocalIPs()
	if len(ips) > 0 {
		ip = ips[0].IP.String()
	} else {
		s.log.ERROR.Printf("couldn't determine ip address- specify %s to override", sempBaseURLEnv)
	}

	uri := fmt.Sprintf("http://%s:%d", ip, s.port)

	return uri
}

func (s *SEMP) handlers(router *mux.Router) {
	sempRouter := router.PathPrefix(basePath).Subrouter()
	getRouter := sempRouter.Methods(http.MethodGet).Subrouter()

	// get description / root / info / status
	getRouter.HandleFunc("/description.xml", s.gatewayDescription)
	getRouter.HandleFunc("/", s.deviceRootHandler)
	getRouter.HandleFunc("/DeviceInfo", s.deviceInfoQuery)
	getRouter.HandleFunc("/DeviceStatus", s.deviceStatusQuery)
	getRouter.HandleFunc("/PlanningRequest", s.devicePlanningQuery)

	// post control messages
	postRouter := sempRouter.Methods(http.MethodPost).Subrouter()
	postRouter.HandleFunc("/", s.deviceControlHandler)
}

func (s *SEMP) writeXML(w http.ResponseWriter, msg any) {
	s.log.TRACE.Printf("send: %+v", msg)

	b, err := xml.MarshalIndent(msg, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write(b)
}

func (s *SEMP) gatewayDescription(w http.ResponseWriter, r *http.Request) {
	uid := "uuid:" + s.uid

	msg := DeviceDescription{
		Xmlns:       urnUPNPDevice,
		SpecVersion: SpecVersion{Major: 1},
		Device: Device{
			DeviceType:      sempGateway,
			FriendlyName:    "evcc",
			Manufacturer:    "evcc.io",
			ModelName:       serverName,
			PresentationURL: s.hostURI,
			UDN:             uid,
			ServiceDefinition: ServiceDefinition{
				Xmlns:          urnSEMPService,
				Server:         s.hostURI,
				BasePath:       basePath,
				Transport:      "HTTP/Pull",
				ExchangeFormat: "XML",
				WsVersion:      "1.1.0",
			},
		},
	}

	s.writeXML(w, msg)
}

func (s *SEMP) deviceRootHandler(w http.ResponseWriter, r *http.Request) {
	msg := Device2EMMsg()
	msg.DeviceInfo = append(msg.DeviceInfo, s.allDeviceInfo()...)
	msg.DeviceStatus = append(msg.DeviceStatus, s.allDeviceStatus()...)
	msg.PlanningRequest = append(msg.PlanningRequest, s.allPlanningRequest()...)
	s.writeXML(w, msg)
}

// deviceInfoQuery answers /semp/DeviceInfo
func (s *SEMP) deviceInfoQuery(w http.ResponseWriter, r *http.Request) {
	msg := Device2EMMsg()

	did := r.URL.Query().Get("DeviceId")
	if did == "" {
		msg.DeviceInfo = append(msg.DeviceInfo, s.allDeviceInfo()...)
	} else {
		for id, lp := range s.site.Loadpoints() {
			if did != s.deviceID(id) {
				continue
			}

			msg.DeviceInfo = append(msg.DeviceInfo, s.deviceInfo(id, lp))
		}

		if len(msg.DeviceInfo) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	s.writeXML(w, msg)
}

// deviceStatusQuery answers /semp/DeviceStatus
func (s *SEMP) deviceStatusQuery(w http.ResponseWriter, r *http.Request) {
	msg := Device2EMMsg()

	did := r.URL.Query().Get("DeviceId")
	if did == "" {
		msg.DeviceStatus = append(msg.DeviceStatus, s.allDeviceStatus()...)
	} else {
		for id, lp := range s.site.Loadpoints() {
			if did != s.deviceID(id) {
				continue
			}

			msg.DeviceStatus = append(msg.DeviceStatus, s.deviceStatus(id, lp))
		}

		if len(msg.DeviceStatus) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	s.writeXML(w, msg)
}

// devicePlanningQuery answers /semp/PlanningRequest
func (s *SEMP) devicePlanningQuery(w http.ResponseWriter, r *http.Request) {
	msg := Device2EMMsg()

	did := r.URL.Query().Get("DeviceId")
	if did == "" {
		msg.PlanningRequest = append(msg.PlanningRequest, s.allPlanningRequest()...)
	} else {
		for id, lp := range s.site.Loadpoints() {
			if did != s.deviceID(id) {
				continue
			}

			if pr := s.planningRequest(id, lp); len(pr.Timeframe) > 0 {
				msg.PlanningRequest = append(msg.PlanningRequest, pr)
			}
		}
	}

	s.writeXML(w, msg)
}

func (s *SEMP) serialNumber(id int) string {
	uidParts := strings.SplitN(s.uid, "-", 5)
	ser := uidParts[len(uidParts)-1]

	return fmt.Sprintf(sempSerialNumber, ser, id)
}

// UniqueDeviceID creates a 6-bytes base device id from machine id
func UniqueDeviceID() ([]byte, error) {
	bytes := 6

	mid := machine.ProtectedID("evcc-semp")

	b, err := hex.DecodeString(mid)
	if err != nil {
		return nil, err
	}

	for i, v := range b {
		b[i%bytes] += v
	}

	return b[:bytes], nil
}

// deviceID combines base device id with device number
func (s *SEMP) deviceID(id int) string {
	// numerically add device number
	did := append([]byte{0, 0}, s.did...)
	return fmt.Sprintf(sempDeviceId, s.vid, ^uint64(0xffff<<48)&(binary.BigEndian.Uint64(did)+uint64(id)))
}

func (s *SEMP) deviceInfo(id int, lp loadpoint.API) DeviceInfo {
	method := MethodEstimation
	if lp.HasChargeMeter() {
		method = MethodMeasurement
	}

	res := DeviceInfo{
		Identification: Identification{
			DeviceID:     s.deviceID(id),
			DeviceName:   lp.GetTitle(),
			DeviceType:   sempCharger,
			DeviceSerial: s.serialNumber(id),
			DeviceVendor: "github.com/evcc-io/evcc",
		},
		Capabilities: Capabilities{
			CurrentPowerMethod:   method,
			InterruptionsAllowed: true,
			OptionalEnergy:       true,
		},
		Characteristics: Characteristics{
			MinPowerConsumption: int(lp.EffectiveMinPower()),
			MaxPowerConsumption: int(lp.EffectiveMaxPower()),
		},
	}

	return res
}

func (s *SEMP) allDeviceInfo() (res []DeviceInfo) {
	for id, lp := range s.site.Loadpoints() {
		res = append(res, s.deviceInfo(id, lp))
	}

	return res
}

func (s *SEMP) deviceStatus(id int, lp loadpoint.API) DeviceStatus {
	chargePower := lp.GetChargePower()

	status := lp.GetStatus()

	deviceStatus := StatusOff
	if status == api.StatusC {
		deviceStatus = StatusOn
	}

	res := DeviceStatus{
		DeviceID:          s.deviceID(id),
		EMSignalsAccepted: false,
		PowerInfo: PowerInfo{
			AveragePower:      int(chargePower),
			AveragingInterval: 60,
		},
		Status: deviceStatus,
	}

	return res
}

func (s *SEMP) allDeviceStatus() (res []DeviceStatus) {
	for id, lp := range s.site.Loadpoints() {
		res = append(res, s.deviceStatus(id, lp))
	}

	return res
}

func (s *SEMP) planningRequest(id int, lp loadpoint.API) (res PlanningRequest) {
	mode := lp.GetMode()
	charging := lp.GetStatus() == api.StatusC
	connected := charging || lp.GetStatus() == api.StatusB

	// remaining max demand duration in seconds
	chargeRemainingDuration := lp.GetRemainingDuration()
	latestEnd := int(chargeRemainingDuration / time.Second)
	if mode == api.ModeMinPV || mode == api.ModePV || latestEnd <= 0 {
		latestEnd = 24 * 3600
	}

	// remaining max energy demand in Wh
	chargeRemainingEnergy := lp.GetRemainingEnergy() * 1e3
	maxEnergy := int(chargeRemainingEnergy)

	// add 1kWh in case we're charging but battery claims full
	if charging && maxEnergy == 0 {
		maxEnergy = 1e3 // 1kWh
	}

	minEnergy := maxEnergy
	if mode == api.ModePV {
		minEnergy = 0
	}

	maxPowerConsumption := int(lp.EffectiveMaxPower())
	minPowerConsumption := int(lp.EffectiveMinPower())
	if mode == api.ModeNow {
		minPowerConsumption = maxPowerConsumption
	}

	if mode != api.ModeOff && connected && maxEnergy > 0 {
		res = PlanningRequest{
			Timeframe: []Timeframe{{
				DeviceID:            s.deviceID(id),
				EarliestStart:       0,
				LatestEnd:           latestEnd,
				MinEnergy:           &minEnergy,
				MaxEnergy:           &maxEnergy,
				MaxPowerConsumption: &maxPowerConsumption,
				MinPowerConsumption: &minPowerConsumption,
			}},
		}
	}

	return res
}

func (s *SEMP) allPlanningRequest() (res []PlanningRequest) {
	for id, lp := range s.site.Loadpoints() {
		if pr := s.planningRequest(id, lp); len(pr.Timeframe) > 0 {
			res = append(res, pr)
		}
	}

	return res
}

func (s *SEMP) deviceControlHandler(w http.ResponseWriter, r *http.Request) {
	var msg EM2Device

	if err := xml.NewDecoder(r.Body).Decode(&msg); err == nil {
		s.log.TRACE.Printf("recv: %+v", msg)
	} else {
		s.log.ERROR.Printf("recv: %+v", msg)
	}

	// ignore control requests
	w.WriteHeader(http.StatusBadRequest)
}
