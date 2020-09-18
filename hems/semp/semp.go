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
	sempBaseUrlEnv   = "SEMP_BASE_URL"
	ssdpMulticast    = "239.255.255.250:1900"
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
	log    *util.Logger
	cache  *util.Cache
	closeC chan struct{}
	doneC  chan struct{}
	uid    string
	port   int
	site   site
}

// site is the minimal interface for accessing site methods
type site interface {
	Configuration() core.SiteConfiguration
	LoadPoints() []*core.LoadPoint
}

// New generates SEMP Gateway listening at /semp endpoint
func New(site site, cache *util.Cache, httpd *server.HTTPd) (*SEMP, error) {
	uid, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("semp")
	ssdp.Logger = log.TRACE

	s := &SEMP{
		doneC: make(chan struct{}),
		log:   log,
		cache: cache,
		site:  site,
		uid:   uid.String(),
	}

	// find external port
	_, port, err := net.SplitHostPort(httpd.Addr)
	if err == nil {
		s.port, err = strconv.Atoi(port)
	}

	s.handlers(httpd.Router)

	return s, err
}

func (s *SEMP) advertise(st, usn string) *ssdp.Advertiser {
	descriptor := s.callbackURI() + basePath + "/description.xml"
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
	if uri := os.Getenv(sempBaseUrlEnv); uri != "" {
		return strings.TrimSuffix(uri, "/")
	}

	ip := "localhost"
	ips := LocalIPs()
	if len(ips) > 0 {
		ip = ips[0].String()
	} else {
		s.log.ERROR.Printf("couldn't determine ip address- specify %s to override", sempBaseUrlEnv)
	}

	uri := fmt.Sprintf("http://%s:%d", ip, s.port)
	s.log.WARN.Printf("%s unspecified, using %s instead", sempBaseUrlEnv, uri)

	return uri
}

func (s *SEMP) handlers(router *mux.Router) {
	sempRouter := router.PathPrefix(basePath).Subrouter()

	sempRouter.HandleFunc("/description.xml", s.gatewayDescription)

	// get root / info / status
	sempRouter.HandleFunc("/", s.deviceRootHandler)
	sempRouter.HandleFunc("/DeviceInfo", s.deviceInfoQuery)
	sempRouter.HandleFunc("/DeviceStatus", s.deviceStatusQuery)
	sempRouter.HandleFunc("/PlanningRequest", s.devicePlanningQuery)
}

func (s *SEMP) writeXML(w http.ResponseWriter, msg interface{}) {
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
	msg := DeviceDescription(sempGateway, serverName, uid, s.callbackURI(), basePath)
	s.writeXML(w, msg)
}

func (s *SEMP) deviceRootHandler(w http.ResponseWriter, r *http.Request) {
	msg := Device2EMMsg()
	msg.DeviceInfo = append(msg.DeviceInfo, s.allDeviceInfo()...)
	msg.DeviceStatus = append(msg.DeviceStatus, s.allDeviceStatus()...)
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
	s.writeXML(w, msg)
}

func (s *SEMP) serialNumber() string {
	uidParts := strings.SplitN(s.uid, "-", 5)
	return uidParts[len(uidParts)-1]
}

func (s *SEMP) deviceID(id int) string {
	return fmt.Sprintf(sempLocalDevice, s.serialNumber(), id)
}

func (s *SEMP) deviceInfo(id int, lp *core.LoadPoint) DeviceInfo {
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
			CurrentPower: CurrentPower{
				Method: method,
			},
			Interruptions: Interruptions{
				InterruptionsAllowed: false,
			},
			Requests: Requests{
				OptionalEnergy: true,
			},
		},
		Characteristics: Characteristics{
			MinPowerConsumption: 230 * int(lp.MinCurrent),
			MaxPowerConsumption: 230 * int(lp.Phases*lp.MaxCurrent),
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

func (s *SEMP) deviceStatus(id int, lp *core.LoadPoint) DeviceStatus {
	var pid util.Param

	var chargePower float64
	chargePowerP := s.cache.Get(pid.UniqueID("chargePower", &id))
	if !chargePowerP.IsNil() {
		chargePower = chargePowerP.Val.(float64)
	}

	status := StatusOff
	statusP := s.cache.Get(pid.UniqueID("status", &id))
	if !statusP.IsNil() {
		if statusP.Val.(string) == string(api.StatusC) {
			status = StatusOn
		}
	}

	res := DeviceStatus{
		DeviceID:          s.deviceID(id),
		EMSignalsAccepted: false,
		PowerConsumption: PowerConsumption{
			PowerInfo: PowerInfo{
				AveragePower:      int(chargePower),
				AveragingInterval: 60,
			},
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
