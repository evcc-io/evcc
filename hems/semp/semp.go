package semp

import (
	"encoding/xml"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/core"
	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/koron/go-ssdp"
)

const (
	sempController   = "Sunny Home Manager"
	sempBaseURLEnv   = "SEMP_BASE_URL"
	sempGateway      = "urn:schemas-simple-energy-management-protocol:device:Gateway:1"
	sempLocalDevice  = "F-28081973-%s-%.02d"
	sempSerialNumber = "%s-%d"
	sempCharger      = "EVCharger"
	basePath         = "/semp"
	maxAge           = 1800
)

var (
	serverName = "EVCC SEMP Server " + server.Version
)

// SEMP is the SMA SEMP server
type SEMP struct {
	log          *util.Logger
	cache        *util.Cache
	closeC       chan struct{}
	doneC        chan struct{}
	controllable bool
	uid          string
	hostURI      string
	port         int
	site         core.SiteAPI
}

// New generates SEMP Gateway listening at /semp endpoint
func New(conf map[string]interface{}, site core.SiteAPI, cache *util.Cache, httpd *server.HTTPd) (*SEMP, error) {
	cc := struct {
		AllowControl bool
	}{}

	if err := util.DecodeOther(conf, &cc); err != nil {
		return nil, err
	}

	uid, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	s := &SEMP{
		doneC:        make(chan struct{}),
		log:          util.NewLogger("semp"),
		cache:        cache,
		site:         site,
		uid:          uid.String(),
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

func (s *SEMP) advertise(st, usn string) *ssdp.Advertiser {
	descriptor := s.hostURI + basePath + "/description.xml"
	ad, err := ssdp.Advertise(st, usn, descriptor, serverName, maxAge)
	if err != nil {
		s.log.ERROR.Println(err)
	}
	return ad
}

// Run executes the SEMP runtime
func (s *SEMP) Run() {
	if s.closeC != nil {
		panic("already running")
	}
	s.closeC = make(chan struct{})

	uid := "uuid:" + s.uid
	ads := []*ssdp.Advertiser{
		s.advertise(ssdp.RootDevice, uid+"::"+ssdp.RootDevice),
		s.advertise(uid, uid),
		s.advertise(sempGateway, uid+"::"+sempGateway),
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
			Manufacturer:    "github.com/andig/evcc",
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

func (s *SEMP) serialNumber() string {
	uidParts := strings.SplitN(s.uid, "-", 5)
	return uidParts[len(uidParts)-1]
}

func (s *SEMP) deviceID(id int) string {
	return fmt.Sprintf(sempLocalDevice, s.serialNumber(), id)
}

func (s *SEMP) deviceInfo(id int, lp core.LoadPointAPI) DeviceInfo {
	method := MethodEstimation
	if lp.HasChargeMeter() {
		method = MethodMeasurement
	}

	res := DeviceInfo{
		Identification: Identification{
			DeviceID:     s.deviceID(id),
			DeviceName:   lp.Name(),
			DeviceType:   sempCharger,
			DeviceSerial: fmt.Sprintf(sempSerialNumber, s.serialNumber(), id),
			DeviceVendor: "github.com/andig/evcc",
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

func (s *SEMP) deviceStatus(id int, lp core.LoadPointAPI) DeviceStatus {
	var chargePower float64
	if chargePowerP, err := s.cache.GetChecked(id, "chargePower"); err == nil {
		chargePower = chargePowerP.Val.(float64)
	}

	isPV := false
	if modeP, err := s.cache.GetChecked(id, "mode"); err == nil {
		if mode, ok := modeP.Val.(api.ChargeMode); ok && (mode == api.ModeMinPV || mode == api.ModePV) {
			isPV = true
		}
	}

	status := StatusOff
	if statusP, err := s.cache.GetChecked(id, "charging"); err == nil {
		if statusP.Val.(bool) {
			status = StatusOn
		}
	}

	var hasVehicle bool
	if hasVehicleP, err := s.cache.GetChecked(id, "hasVehicle"); err == nil {
		hasVehicle = hasVehicleP.Val.(bool)
	}

	res := DeviceStatus{
		DeviceID:          s.deviceID(id),
		EMSignalsAccepted: s.controllable && isPV && hasVehicle,
		PowerInfo: PowerInfo{
			AveragePower:      int(chargePower),
			AveragingInterval: 60,
		},
		Status: status,
	}

	return res
}

func (s *SEMP) allDeviceStatus() (res []DeviceStatus) {
	for id, lp := range s.site.LoadPoints() {
		res = append(res, s.deviceStatus(id, lp))
	}

	return res
}

func (s *SEMP) planningRequest(id int, lp core.LoadPointAPI) (res PlanningRequest) {
	mode := api.ModeOff
	if modeP, err := s.cache.GetChecked(id, "mode"); err == nil {
		mode = modeP.Val.(api.ChargeMode)
	}

	var connected bool
	if connectedP, err := s.cache.GetChecked(id, "connected"); err == nil {
		connected = connectedP.Val.(bool)
	}

	var charging bool
	if chargingP, err := s.cache.GetChecked(id, "charging"); err == nil {
		charging = chargingP.Val.(bool)
	}

	chargeEstimate := time.Duration(-1)
	if chargeEstimateP, err := s.cache.GetChecked(id, "chargeEstimate"); err == nil {
		chargeEstimate = chargeEstimateP.Val.(time.Duration)
	}

	latestEnd := int(chargeEstimate / time.Second)
	if mode == api.ModeMinPV || mode == api.ModePV || latestEnd <= 0 {
		latestEnd = 24 * 3600
	}

	var maxEnergy int
	if chargeRemainingEnergyP, err := s.cache.GetChecked(id, "chargeRemainingEnergy"); err == nil {
		maxEnergy = int(chargeRemainingEnergyP.Val.(float64))
	}

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

	if connected && maxEnergy > 0 {
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

			demand := core.RemoteSoftDisable
			if dev.On {
				demand = core.RemoteEnable
			}

			lp.RemoteControl(sempController, demand)
		}
	}

	w.WriteHeader(http.StatusOK)
}
