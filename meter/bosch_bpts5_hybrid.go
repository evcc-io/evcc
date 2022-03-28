package meter

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// Bosch is the Bosch BPT-S 5 Hybrid meter
type BoschBpts5HybridApiClient struct {
	*request.Helper
	uri, wuSid             string
	currentBatterySocValue float64
	einspeisung            float64
	strombezugAusNetz      float64
	pvLeistungWatt         float64
	batterieLadeStrom      float64
	verbrauchVonBatterie   float64
	logger                 *util.Logger
}

type BoschBpts5Hybrid struct {
	usage                   string
	currentErr              error
	currentTotalEnergyValue float64
	requestClient           *BoschBpts5HybridApiClient
	logger                  *util.Logger
}

var boschInstance *BoschBpts5HybridApiClient = nil

func init() {
	registry.Add("bosch-bpts5-hybrid", NewBoschBpts5HybridFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateBoschBpts5Hybrid -b api.Meter -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,SoC,func() (float64, error)"

// NewBoschBpts5HybridFromConfig creates a Bosch BPT-S 5 Hybrid Meter from generic config
func NewBoschBpts5HybridFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI, Usage string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}

	_, err := url.Parse(cc.URI)
	if err != nil {
		return nil, fmt.Errorf("%s is invalid: %s", cc.URI, err)
	}

	return NewBoschBpts5Hybrid(cc.URI, cc.Usage)
}

// NewBoschBpts5Hybrid creates a Bosch BPT-S 5 Hybrid Meter
func NewBoschBpts5Hybrid(uri, usage string) (api.Meter, error) {
	log := util.NewLogger("bosch")

	if boschInstance == nil {
		boschInstance = &BoschBpts5HybridApiClient{
			Helper:                 request.NewHelper(log),
			uri:                    util.DefaultScheme(strings.TrimSuffix(uri, "/"), "http"),
			currentBatterySocValue: 0.0,
			einspeisung:            0.0,
			strombezugAusNetz:      0.0,
			pvLeistungWatt:         0.0,
			batterieLadeStrom:      0.0,
			verbrauchVonBatterie:   0.0,
			logger:                 log,
		}

		// ignore the self signed certificate
		boschInstance.Client.Transport = request.NewTripper(log, transport.Insecure())
		// create cookie jar to save login tokens
		boschInstance.Client.Jar, _ = cookiejar.New(nil)

		if err := boschInstance.Login(); err != nil {
			return nil, err
		}

		go readLoop(boschInstance)
	}

	m := &BoschBpts5Hybrid{
		usage:                   strings.ToLower(usage),
		currentErr:              nil,
		currentTotalEnergyValue: 0.0,
		requestClient:           boschInstance,
		logger:                  log,
	}

	// decorate api.MeterEnergy
	var totalEnergy func() (float64, error)
	if m.usage == "grid" || m.usage == "pv" {
		totalEnergy = m.totalEnergy
	}

	// decorate api.BatterySoC
	var batterySoC func() (float64, error)
	if usage == "battery" {
		batterySoC = m.batterySoC
	}

	return decorateBoschBpts5Hybrid(m, totalEnergy, batterySoC), nil
}

// Login calls login and saves the returned cookie
func (m *BoschBpts5HybridApiClient) Login() error {
	resp, err := m.Client.Get(m.uri)

	if err != nil {
		m.logger.ERROR.Println("Error during login: first GET", err)
		return err
	}

	if resp.StatusCode >= 300 {
		errorText := "Error while getting WUI SID. Response code was >=300:"
		m.logger.ERROR.Println(errorText)
		return errors.New(errorText)
	}

	defer resp.Body.Close()

	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		m.logger.ERROR.Println("Error during login: read response body", err)
		return err
	}

	err = extractWuiSidFromBody(m, string(body))

	if err != nil {
		m.logger.ERROR.Println("Error during login: error extract WUI SID", err)
		return err
	}

	return nil
}

func readLoop(m *BoschBpts5HybridApiClient) {
	for {
		loopError := executeRead(m)

		if loopError != nil {
			m.logger.ERROR.Println("error during read loop. Try to re-login/get new WUI SID")
			m.Login()
		}

		time.Sleep(5000 * time.Millisecond)
	}
}

func executeRead(m *BoschBpts5HybridApiClient) error {
	var postMessge = []byte(`action=get.hyb.overview&flow=1`)
	resp, err := m.Client.Post(m.uri+"/cgi-bin/ipcclient.fcgi?"+m.wuSid, "text/plain", bytes.NewBuffer(postMessge))

	if err != nil {
		m.logger.ERROR.Println("Error during data retrieval request: POST", err)
		return err
	}

	defer resp.Body.Close()

	//Read the response body
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		m.logger.ERROR.Println("Error during data retrieval request: read body", err)
		return err
	}

	if resp.StatusCode >= 300 {
		errorText := "error while reading values. response code was >=300:"
		m.logger.ERROR.Println(errorText)
		return errors.New(errorText)
	}

	sb := string(body)
	return extractValues(m, sb)
}

func parseWattValue(inputString string) (float64, error) {
	if len(strings.TrimSpace(inputString)) == 0 || strings.Contains(inputString, "nbsp;") {
		return 0.0, nil
	}

	zahlenString := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(inputString, "kW", " "), "von", " "))

	resultFloat, err := strconv.ParseFloat(zahlenString, 64)

	return resultFloat * 1000.0, err
}

func extractValues(m *BoschBpts5HybridApiClient, body string) error {
	if strings.Contains(body, "session invalid") {
		m.logger.DEBUG.Println("extractValues: Session invalid. Performing Re-login")
		m.Login()
		return nil
	}

	values := strings.Split(body, "|")

	soc, err := strconv.Atoi(values[3])

	if err != nil {
		m.logger.ERROR.Println("extractValues: error during value parsing 1", err)
		return err
	}

	m.currentBatterySocValue = float64(soc)
	m.einspeisung, err = parseWattValue(values[11])

	if err != nil {
		m.logger.ERROR.Println("extractValues: error during value parsing 2", err)
		return err
	}

	m.strombezugAusNetz, err = parseWattValue(values[14])

	if err != nil {
		m.logger.ERROR.Println("extractValues: error during value parsing 3", err)
		return err
	}

	m.pvLeistungWatt, err = parseWattValue(values[2])

	if err != nil {
		m.logger.ERROR.Println("extractValues: error during value parsing 4", err)
		return err
	}

	m.batterieLadeStrom, err = parseWattValue(values[10])

	if err != nil {
		m.logger.ERROR.Println("extractValues: error during value parsing 5", err)
		return err
	}

	m.verbrauchVonBatterie, err = parseWattValue(values[13])

	m.logger.DEBUG.Println("extractValues: batterieLadeStrom=", m.batterieLadeStrom, ";currentBatterySocValue=", m.currentBatterySocValue, ";einspeisung=", m.einspeisung, ";pvLeistungWatt=", m.pvLeistungWatt, ";strombezugAusNetz=", m.strombezugAusNetz, ";verbrauchVonBatterie=", m.verbrauchVonBatterie)

	return err
}

func extractWuiSidFromBody(m *BoschBpts5HybridApiClient, body string) error {
	index := strings.Index(body, "WUI_SID=")

	if index < 0 {
		m.wuSid = ""
		m.logger.ERROR.Println("Error while extracting WUI_SID. Body was= " + body)
		return errors.New("Error while extracting WUI_SID. Body was= " + body)
	}

	m.wuSid = body[index+9 : index+9+15]

	m.logger.DEBUG.Println("extractWuiSidFromBody: result=", m.wuSid)

	return nil
}

// CurrentPower implements the api.Meter interface
func (m *BoschBpts5Hybrid) CurrentPower() (float64, error) {
	if m.usage == "grid" {
		if m.requestClient.einspeisung > 0.0 {
			return -1.0 * m.requestClient.einspeisung, nil
		} else {
			return m.requestClient.strombezugAusNetz, nil
		}
	}
	if m.usage == "pv" {
		return m.requestClient.pvLeistungWatt, nil
	}
	if m.usage == "battery" {
		if m.requestClient.batterieLadeStrom > 0.0 {
			return -1.0 * m.requestClient.batterieLadeStrom, nil
		} else {
			return m.requestClient.verbrauchVonBatterie, nil
		}
	}
	return 0.0, nil
}

// totalEnergy implements the api.MeterEnergy interface
func (m *BoschBpts5Hybrid) totalEnergy() (float64, error) {
	return m.currentTotalEnergyValue, nil
}

// batterySoC implements the api.Battery interface
func (m *BoschBpts5Hybrid) batterySoC() (float64, error) {
	return m.requestClient.currentBatterySocValue, nil
}
