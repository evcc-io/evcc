package globalconfig

import (
	"encoding/json"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/hems/shm"
	"github.com/evcc-io/evcc/plugin/mqtt"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/modbus"
)

type All struct {
	Network      Network
	Log          string
	SponsorToken string
	Plant        string // telemetry plant id
	Telemetry    bool
	Mcp          bool
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
	SHM          shm.Config
	Messaging    Messaging
	Meters       []config.Named
	Chargers     []config.Named
	Vehicles     []config.Named
	Tariffs      Tariffs
	Site         map[string]any
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
	Port            int    `json:"port"`
	ReadOnly        string `yaml:",omitempty" json:"readonly,omitempty"`
	modbus.Settings `mapstructure:",squash" yaml:",inline,omitempty" json:"settings,omitempty"`
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
	mqtt.Config `mapstructure:",squash"`
	Topic       string `json:"topic"`
}

// Redacted implements the redactor interface used by the tee publisher
func (m Mqtt) Redacted() any {
	return Mqtt{
		Config: mqtt.Config{
			Broker:     m.Broker,
			User:       m.User,
			Password:   masked(m.Password),
			ClientID:   m.ClientID,
			Insecure:   m.Insecure,
			CaCert:     masked(m.CaCert),
			ClientCert: masked(m.ClientCert),
			ClientKey:  masked(m.ClientKey),
		},
		Topic: m.Topic,
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
	return Influx{
		URL:      c.URL,
		Database: c.Database,
		Token:    masked(c.Token),
		Org:      c.Org,
		User:     c.User,
		Password: masked(c.Password),
		Insecure: c.Insecure,
	}
}

type DB struct {
	Type string
	Dsn  string
}

type Messaging struct {
	Events   map[string]MessagingEventTemplate
	Services []config.Typed
}

// MessagingEventTemplate is the push message configuration for an event
type MessagingEventTemplate struct {
	Title, Msg string
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
	Schema_     string `json:"schema,omitempty" mapstructure:"schema"` // TODO deprecated
	ExternalUrl string `json:"externalUrl"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
}

func (c Network) HostPort() string {
	host := "localhost"
	if h, err := os.Hostname(); err == nil {
		host = h
	}
	if ips := util.LocalIPs(); len(ips) > 0 {
		host = ips[0].IP.String()
	}
	if c.Port == 80 {
		return host
	}
	return net.JoinHostPort(host, strconv.Itoa(c.Port))
}

func (c Network) InternalURL() string {
	return "http://" + c.HostPort()
}

func (c Network) ExternalURL() string {
	if c.ExternalUrl != "" {
		return strings.TrimRight(c.ExternalUrl, "/")
	}
	return c.InternalURL()
}

// MarshalJSON includes the computed InternalUrl field in JSON output
func (c Network) MarshalJSON() ([]byte, error) {
	type networkAlias Network
	return json.Marshal(struct {
		networkAlias
		InternalUrl string `json:"internalUrl"`
	}{
		networkAlias: networkAlias(c),
		InternalUrl:  c.InternalURL(),
	})
}
