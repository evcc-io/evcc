// Package lgpcs implements access to the LG pcs device (aka inverter).
// Pcs is the LG power conditioning system that converts the PV (or battery) - DC current into AC current (and controls the batteries)
package lgpcs

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/cast"
)

type Com struct {
	*request.Helper
	uri      string // URI address of the LG ESS inverter - e.g. "https://192.168.1.28"
	password string // registration number of the LG ESS Inverter - e.g. "DE2001..."
	authKey  string // auth_key returned during login and renewed with new login after expiration
	essType  Model  // currently the LG Ess Home 8/10 and Home 15 are supported
	dataG    func() (EssData, error)
}

var (
	once     sync.Once
	instance *Com
)

// GetInstance implements the singleton pattern to handle the access via the authkey to the PCS of the LG ESS HOME system
func GetInstance(uri, registration, password string, cache time.Duration, essType Model) (*Com, error) {
	uri = util.DefaultScheme(strings.TrimSuffix(uri, "/"), "https")

	var err error
	once.Do(func() {
		log := util.NewLogger("lgess")
		instance = &Com{
			Helper:   request.NewHelper(log),
			uri:      uri,
			password: password,
			essType:  essType,
		}

		if registration != "" {
			instance.password = registration
		}

		// ignore the self signed certificate
		instance.Client.Transport = request.NewTripper(log, transport.Insecure())

		// caches the data access for the "cache" time duration
		// sends a new request to the pcs if the cache is expired and Data() requested
		instance.dataG = util.Cached(instance.essInfo, cache)

		// do first login if no authKey exists and uri and password exist
		if instance.authKey == "" && instance.uri != "" && instance.password != "" {
			err = instance.Login()
		}
	})

	// check if both password and registration are provided
	if password != "" && registration != "" {
		return nil, errors.New("cannot have registration and password")
	}

	// check if different uris are provided
	if uri != "" && instance.uri != uri {
		return nil, fmt.Errorf("uri mismatch: %s vs %s", instance.uri, uri)
	}

	// check if different passwords are provided
	if password != "" && instance.password != password {
		return nil, errors.New("password mismatch")
	}

	return instance, err
}

// Login calls login and stores the returned authorization key
func (m *Com) Login() error {
	data := map[string]interface{}{
		"password": m.password,
	}

	uri := fmt.Sprintf("%s/v1/user/setting/login", m.uri)
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

	// try to login as ESS installer if user login failed
	if res.Status == "password mismatched" {
		uri := fmt.Sprintf("%s/v1/login", m.uri)
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

		if res.Status != "success" {
			return fmt.Errorf("login failed: %s", res.Status)
		}

		// read auth_key from response
		m.authKey = res.AuthKey

		return nil
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
	f := func(body io.ReadSeeker) (*http.Request, error) {
		uri := fmt.Sprintf("%s/v1/user/essinfo/home", m.uri)
		return request.New(http.MethodPost, uri, body, request.JSONEncoding)
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

// BatteryMode sets the battery mode
func (m *Com) BatteryMode(mode string, soc int, autocharge bool) error {
	var res struct{}
	return m.request(func(body io.ReadSeeker) (*http.Request, error) {
		uri := fmt.Sprintf("%s/v1/user/setting/batt", m.uri)
		return request.New(http.MethodPut, uri, body, request.JSONEncoding)
	}, map[string]string{
		"backupmode": mode,
		"backup_soc": strconv.Itoa(soc),
		"autocharge": strconv.Itoa(cast.ToInt(autocharge)),
	}, &res)
}

func (m *Com) request(f func(io.ReadSeeker) (*http.Request, error), payload any, meterData any) error {
	data := map[string]string{
		"auth_key": m.authKey,
	}

	if err := mapstructure.Decode(payload, &data); err != nil {
		return err
	}

	req, err := f(request.MarshalJSON(data))
	if err != nil {
		return err
	}

	if err := m.DoJSON(req, &meterData); err != nil {
		// re-login if request returns 405-error
		if se := new(request.StatusError); errors.As(err, &se) && se.StatusCode() != http.StatusMethodNotAllowed {
			return err
		}

		if err := m.Login(); err != nil {
			return err
		}

		data["auth_key"] = m.authKey
		req, err := f(request.MarshalJSON(data))
		if err != nil {
			return err
		}

		return m.DoJSON(req, &meterData)
	}

	return nil
}
