package globalconfig

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/push"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/modbus"
)

type All struct {
	Network      Network
	Log          string
	SponsorToken string
	Plant        string // telemetry plant id
	Telemetry    bool
	Metrics      bool
	Profile      bool
	Levels       map[string]string
	Interval     time.Duration
	Database     DB
	Mqtt         Mqtt
	ModbusProxy  []ModbusProxy
	Javascript   []Javascript
	Go           []Go
	Influx       Influx
	EEBus        eebus.Config
	HEMS         Hems
	Messaging    Messaging
	Meters       []config.Named
	Chargers     []config.Named
	Vehicles     []config.Named
	Tariffs      Tariffs
	Site         map[string]interface{}
	Loadpoints   []config.Named
	Circuits     []config.Named
}

type Javascript struct {
	VM     string
	Script string
}

type Go struct {
	VM     string
	Script string
}

type ModbusProxy struct {
	Port            int
	ReadOnly        string `yaml:",omitempty" json:",omitempty"`
	modbus.Settings `mapstructure:",squash" yaml:",inline,omitempty" json:",omitempty"`
}

var _ api.Redactor = (*Hems)(nil)

type Hems config.Typed

func (c Hems) Redacted() any {
	return struct {
		Type string `json:"type,omitempty"`
	}{
		Type: c.Type,
	}
}

var _ api.Redactor = (*Mqtt)(nil)

func masked(s any) string {
	if s != "" {
		return "***"
	}
	return ""
}

type Mqtt struct {
	Broker     string `json:"broker"`
	Topic      string `json:"topic"`
	User       string `json:"user"`
	ClientID   string `json:"clientID"`
	Insecure   bool   `json:"insecure"`
	Password   string `json:"password"`
	CaCert     string `json:"caCert"`
	ClientCert string `json:"clientCert"`
	ClientKey  string `json:"clientKey"`
}

// Redacted implements the redactor interface used by the tee publisher
func (m Mqtt) Redacted() any {
	return struct {
		Broker     string `json:"broker"`
		Topic      string `json:"topic"`
		User       string `json:"user,omitempty"`
		ClientID   string `json:"clientID,omitempty"`
		Insecure   bool   `json:"insecure,omitempty"`
		Password   string `json:"password,omitempty"`
		CaCert     string `json:"caCert,omitempty"`
		ClientCert string `json:"clientCert,omitempty"`
		ClientKey  string `json:"clientKey,omitempty"`
	}{
		Broker:     m.Broker,
		Topic:      m.Topic,
		User:       m.User,
		ClientID:   m.ClientID,
		Insecure:   m.Insecure,
		Password:   masked(m.Password),
		CaCert:     masked(m.CaCert),
		ClientCert: masked(m.ClientCert),
		ClientKey:  masked(m.ClientKey),
	}
}

// Influx is the influx db configuration
type Influx struct {
	URL      string `json:"url"`
	Database string `json:"database"`
	Token    string `json:"token"`
	Org      string `json:"org"`
	User     string `json:"user"`
	Password string `json:"password"`
	Insecure bool   `json:"insecure"`
}

// Redacted implements the redactor interface used by the tee publisher
func (c Influx) Redacted() any {
	return struct {
		URL      string `json:"url"`
		Database string `json:"database"`
		Org      string `json:"org"`
		User     string `json:"user"`
		Insecure bool   `json:"insecure"`
		Password string `json:"password,omitempty"`
		Token    string `json:"token,omitempty"`
	}{
		URL:      c.URL,
		Database: c.Database,
		Org:      c.Org,
		User:     c.User,
		Insecure: c.Insecure,
		Password: masked(c.Password),
		Token:    masked(c.Token),
	}
}

type DB struct {
	Type string
	Dsn  string
}

type Messaging struct {
	Events   map[string]push.EventTemplateConfig
	Services []config.Typed
}

func (c Messaging) Configured() bool {
	return len(c.Services) > 0 || len(c.Events) > 0
}

type Tariffs struct {
	Currency string
	Grid     config.Typed
	FeedIn   config.Typed
	Co2      config.Typed
	Planner  config.Typed
	Solar    []config.Typed
}

type Network struct {
	Schema string `json:"schema"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
}

func (c Network) HostPort() string {
	if c.Schema == "http" && c.Port == 80 || c.Schema == "https" && c.Port == 443 {
		return c.Host
	}
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

func (c Network) URI() string {
	return fmt.Sprintf("%s://%s", c.Schema, c.HostPort())
}
