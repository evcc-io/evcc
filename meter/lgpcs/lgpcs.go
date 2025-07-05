// Package lgpcs implements access to the LG pcs device (aka inverter).
// Pcs is the LG power conditioning system that converts the PV (or battery) - DC current into AC current (and controls the batteries)
package lgpcs

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/spf13/cast"
)

type Com struct {
	*request.Helper
	uri          string // URI address of the LG ESS inverter - e.g. "https://192.168.1.28"
	password     string // user password, usually MAC address of the LG ESS in lowercase without colons
	registration string // registration number of the LG ESS Inverter - e.g. "DE2001..."
	authKey      string // auth_key returned during login and renewed with new login after expiration
	essType      Model  // currently the LG Ess Home 8/10 and Home 15 are supported
	dataG        func() (EssData, error)
	log          *util.Logger
}

var (
	instances map[string]*Com = map[string]*Com{}
	mu        sync.Mutex      = sync.Mutex{}
)

// GetInstance retrives a singleton per uri from a map to handle the access via the authkey to the PCS of the LG ESS HOME system
func GetInstance(uri, registration, password string, cache time.Duration, essType Model) (*Com, error) {
	mu.Lock()
	defer mu.Unlock()

	uri = util.DefaultScheme(strings.TrimSuffix(uri, "/"), "https")

	instance, found := instances[uri]
	if found {
		return instance, nil
	}

	log := util.NewLogger(fmt.Sprintf("lgess-%s", strings.TrimPrefix("https://", uri)))
	instance = &Com{uri: uri,
		Helper:       request.NewHelper(log),
		registration: registration,
		password:     password,
		essType:      essType,
		log:          log,
	}
	// put instance into the cache map
	instances[uri] = instance

	// ignore the self signed certificate
	instance.Client.Transport = request.NewTripper(log, transport.Insecure())

	// caches the data access for the "cache" time duration
	// sends a new request to the pcs if the cache is expired and Data() requested
	instance.dataG = util.Cached(instance.essInfo, cache)

	// do first login if no authKey exists and uri and password exist
	if instance.authKey == "" && instance.uri != "" && (instance.password != "" || instance.registration != "") {
		if err := instance.Login(); err != nil {
			return nil, err
		}
		return instance, nil
	}

	return nil, errors.New("missing credentials")
}

// Login calls login and stores the returned authorization key
func (m *Com) Login() error {
	// check if at least one of password and registration is provided
	if m.password == "" && m.registration == "" {
		return errors.New("neither registration nor password provided - at least one needed")
	}

	data := map[string]interface{}{
		"password": m.password,
	}
	uri := fmt.Sprintf("%s/v1/user/setting/login", m.uri)

	if m.password == "" { // use installer login
		m.log.DEBUG.Println("installer login")
		uri = fmt.Sprintf("%s/v1/login", m.uri)
		data["password"] = m.registration
	}

	if m.authKey != "" {
		m.log.DEBUG.Println("re-login")
	}

	req, err := request.New(http.MethodPut, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	// read auth_key from response body
	var res struct {
		Status  string `json:"status,omitempty"`
		AuthKey string `json:"auth_key"`
	}

	if err := m.DoJSON(req, &res); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// check login response status
	if res.Status != "success" {
		return fmt.Errorf("login failed: %s", res.Status)
	}

	// read auth_key from response
	m.authKey = res.AuthKey

	return nil
}

// Data gives the cached data read from the pcs.
func (m *Com) Data() (EssData, error) {
	return m.dataG()
}

// essInfo reads essinfo/home
func (m *Com) essInfo() (EssData, error) {
	f := func(payload any) (*http.Request, error) {
		uri := fmt.Sprintf("%s/v1/user/essinfo/home", m.uri)
		return request.New(http.MethodPost, uri, request.MarshalJSON(payload), request.JSONEncoding)
	}

	if m.essType == LgEss8 {
		var res MeterResponse8
		err := m.request(f, nil, &res)
		return res, err
	}

	var res MeterResponse15
	err := m.request(f, nil, &res)
	return res, err
}

func (m *Com) GetSystemInfo() (SystemInfoResponse, error) {
	f := func(payload any) (*http.Request, error) {
		uri := fmt.Sprintf("%s/v1/user/setting/systeminfo", m.uri)
		return request.New(http.MethodPost, uri, request.MarshalJSON(payload), request.JSONEncoding)
	}
	var res SystemInfoResponse
	err := m.request(f, nil, &res)
	return res, err
}

func (m *Com) GetFirmwareVersion() (int, error) {
	systemInfo, err := m.GetSystemInfo()
	if err != nil {
		return 0, err
	}

	// extract the patch number behind a dot that is always followed by at least 4 digits
	if match := regexp.MustCompile(`\.(\d{4})$`).FindStringSubmatch(systemInfo.Version.PMSVersion); len(match) > 1 {
		return strconv.Atoi(match[1])
	}

	return 0, errors.New("firmware version not found")
}

// BatteryMode sets the battery mode
func (m *Com) BatteryMode(mode string, soc int, autocharge bool) error {
	var res struct{}
	return m.request(func(payload any) (*http.Request, error) {
		uri := fmt.Sprintf("%s/v1/user/setting/batt", m.uri)
		return request.New(http.MethodPut, uri, request.MarshalJSON(payload), request.JSONEncoding)
	}, map[string]string{
		"backupmode": mode,
		"backup_soc": strconv.Itoa(soc),
		"autocharge": strconv.Itoa(cast.ToInt(autocharge)),
	}, &res)
}

func (m *Com) request(f func(any) (*http.Request, error), payload map[string]string, res any) error {
	data := map[string]string{
		"auth_key": m.authKey,
	}
	maps.Copy(data, payload)

	req, err := f(data)
	if err != nil {
		return err
	}

	err = m.DoJSON(req, &res)
	if err == nil {
		return nil
	}

	// re-login if request returns 405-error
	if se := new(request.StatusError); errors.As(err, &se) && se.StatusCode() != http.StatusMethodNotAllowed {
		return err
	}

	if err := m.Login(); err != nil {
		return err
	}

	data["auth_key"] = m.authKey
	req, err = f(data)
	if err != nil {
		return err
	}

	return m.DoJSON(req, &res)
}
