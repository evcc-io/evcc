package semp

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
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/koron/go-ssdp"
	"github.com/panta/machineid"
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

var serverName = "EVCC SEMP Server " + server.Version

// SEMP is the SMA SEMP server
type SEMP struct {
	log          *util.Logger
	closeC       chan struct{}
	doneC        chan struct{}
	controllable bool
	vid          string
	did          []byte
	uid          string
	hostURI      string
	port         int
	site         site.API
}

// New generates SEMP Gateway listening at /semp endpoint
func New(conf map[string]interface{}, site site.API, httpd *server.HTTPd) (*SEMP, error) {
	cc := struct {
		VendorID     string
		DeviceID     string
		AllowControl bool
	}{
		VendorID: "28081973",
	}

	if err := util.DecodeOther(conf, &cc); err != nil {
		return nil, err
	}

	uid, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	if len(cc.VendorID) != 8 {
		return nil, fmt.Errorf("invalid vendor id: %v", cc.VendorID)
	}

	var did []byte
	if cc.DeviceID == "" {
		if did, err = UniqueDeviceID(); err != nil {
			return nil, fmt.Errorf("creating device id: %w", err)
		}
	} else {
		if did, err = hex.DecodeString(cc.DeviceID); err != nil {
			return nil, fmt.Errorf("device id: %w", err)
		}
	}

	if len(did) != 6 {
		return nil, fmt.Errorf("invalid device id: %v", cc.DeviceID)
	}

	s := &SEMP{
		doneC:        make(chan struct{}),
		log:          util.NewLogger("semp"),
		site:         site,
		uid:          uid.String(),
		vid:          cc.VendorID,
		did:          did,
		controllable: cc.AllowControl,
	}

	// find external port
	_, port, err := net.SplitHostPort(httpd.Addr)
	if err == nil {
		s.port, err = strconv.Atoi(port)
	}

	s.hostURI = s.callbackURI()

	s.handlers(httpd.Router())

	return s, err
}

func (s *SEMP) advertise(st, usn string) (*ssdp.Advertiser, error) {
	descriptor := s.hostURI + basePath + "/description.xml"
	return ssdp.Advertise(st, usn, descriptor, serverName, maxAge)
}

// Run executes the SEMP runtime
func (s *SEMP) Run() {
	if s.closeC != nil {
		panic("already running")
	}
	s.closeC = make(chan struct{})

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

	ticker := time.NewTicker(maxAge * time.Second / 2)

ANNOUNCE:
	for {
		select {
		case <-ticker.C:
			for _, ad := range ads {
				if err := ad.Alive(); err != nil {
					s.log.ERROR.Println(err)
				}
			}
		case <-s.closeC:
			break ANNOUNCE
		}
	}

	for _, ad := range ads {
		if err := ad.Bye(); err != nil {
			s.log.ERROR.Println(err)
		}
	}

	close(s.doneC)
}

// Stop stops the SEMP runtime
func (s *SEMP) Stop() {
	if s.closeC == nil {
		panic("not running")
	}
	close(s.closeC)
}

// Done returns the done channel. The channel is closed after byebye has been sent.
func (s *SEMP) Done() chan struct{} {
	return s.doneC
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
	s.log.WARN.Printf("%s unspecified, using %s instead", sempBaseURLEnv, uri)

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

func (s *SEMP) writeXML(w http.ResponseWriter, msg interface{}) {
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
			Manufacturer:    "github.com/evcc-io/evcc",
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
		for id, lp := range s.site.LoadPoints() {
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
		for id, lp := range s.site.LoadPoints() {
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
		for id, lp := range s.site.LoadPoints() {
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

	mid, err := machineid.ProtectedID("evcc-semp")
	if err != nil {
		return nil, err
	}

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
			DeviceName:   lp.Name(),
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
			MinPowerConsumption: int(lp.GetMinPower()),
			MaxPowerConsumption: int(lp.GetMaxPower()),
		},
	}

	return res
}

func (s *SEMP) allDeviceInfo() (res []DeviceInfo) {
	for id, lp := range s.site.LoadPoints() {
		res = append(res, s.deviceInfo(id, lp))
	}

	return res
}

func (s *SEMP) deviceStatus(id int, lp loadpoint.API) DeviceStatus {
	chargePower := lp.GetChargePower()

	status := lp.GetStatus()
	mode := lp.GetMode()
	isPV := mode == api.ModeMinPV || mode == api.ModePV

	deviceStatus := StatusOff
	if status == api.StatusC {
		deviceStatus = StatusOn
	}

	connected := status == api.StatusB || status == api.StatusC

	res := DeviceStatus{
		DeviceID:          s.deviceID(id),
		EMSignalsAccepted: s.controllable && isPV && connected,
		PowerInfo: PowerInfo{
			AveragePower:      int(chargePower),
			AveragingInterval: 60,
		},
		Status: deviceStatus,
	}

	return res
}

func (s *SEMP) allDeviceStatus() (res []DeviceStatus) {
	for id, lp := range s.site.LoadPoints() {
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
	chargeRemainingEnergy := lp.GetRemainingEnergy()
	maxEnergy := int(chargeRemainingEnergy)

	// add 1kWh in case we're charging but battery claims full
	if charging && maxEnergy == 0 {
		maxEnergy = 1e3 // 1kWh
	}

	minEnergy := maxEnergy
	if mode == api.ModePV {
		minEnergy = 0
	}

	maxPowerConsumption := int(lp.GetMaxPower())
	minPowerConsumption := int(lp.GetMinPower())
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
	for id, lp := range s.site.LoadPoints() {
		if pr := s.planningRequest(id, lp); len(pr.Timeframe) > 0 {
			res = append(res, pr)
		}
	}

	return res
}

func (s *SEMP) deviceControlHandler(w http.ResponseWriter, r *http.Request) {
	var msg EM2Device

	err := xml.NewDecoder(r.Body).Decode(&msg)
	s.log.TRACE.Printf("recv: %+v", msg)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for _, dev := range msg.DeviceControl {
		did := dev.DeviceID

		for id, lp := range s.site.LoadPoints() {
			if did != s.deviceID(id) {
				continue
			}

			if mode := lp.GetMode(); mode != api.ModeMinPV && mode != api.ModePV {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// ignore requests if not controllable
			if !s.controllable {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			demand := loadpoint.RemoteSoftDisable
			if dev.On {
				demand = loadpoint.RemoteEnable
			}

			lp.RemoteControl(sempController, demand)
		}
	}

	w.WriteHeader(http.StatusOK)
}
