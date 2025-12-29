package snmp

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)

type Handler interface {
	Get(oids []string) (result *gosnmp.SnmpPacket, err error)
}

type Connection struct {
	Handler Handler
}

func NewConnection(uri, version, community string, auth Auth) (*Connection, error) {
	if !strings.Contains(uri, "://") {
		uri = "udp://" + uri
	}
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	host := u.Hostname()
	portStr := u.Port()
	if portStr == "" {
		portStr = "161"
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	g := &gosnmp.GoSNMP{
		Target:    host,
		Port:      uint16(port),
		Transport: u.Scheme,
		Retries:   2,
		Timeout:   time.Duration(2) * time.Second,
	}

	switch strings.ToLower(version) {
	case "1":
		g.Version = gosnmp.Version1
		g.Community = community
	case "3":
		if auth.User == "" {
			return nil, fmt.Errorf("SNMP v3 requires a user")
		}
		g.Version = gosnmp.Version3
		g.SecurityModel = gosnmp.UserSecurityModel
		g.MsgFlags = auth.MsgFlags()
		g.SecurityParameters = &gosnmp.UsmSecurityParameters{
			UserName:                 auth.User,
			AuthenticationProtocol:   auth.GetAuthProtocol(),
			AuthenticationPassphrase: auth.AuthPassword,
			PrivacyProtocol:          auth.GetPrivProtocol(),
			PrivacyPassphrase:        auth.PrivPassword,
		}
	default:
		g.Version = gosnmp.Version2c
		g.Community = community
	}

	if err := g.Connect(); err != nil {
		return nil, err
	}

	return &Connection{Handler: g}, nil
}

type Auth struct {
	User          string
	SecurityLevel string
	AuthType      string `mapstructure:"auth"`
	AuthPassword  string `mapstructure:"authPass"`
	PrivType      string `mapstructure:"priv"`
	PrivPassword  string `mapstructure:"privPass"`
}

func (a Auth) MsgFlags() gosnmp.SnmpV3MsgFlags {
	switch strings.ToLower(a.SecurityLevel) {
	case "authpriv":
		return gosnmp.AuthPriv
	case "authnopriv":
		return gosnmp.AuthNoPriv
	default:
		return gosnmp.NoAuthNoPriv
	}
}

func (a Auth) GetAuthProtocol() gosnmp.SnmpV3AuthProtocol {
	switch strings.ToUpper(a.AuthType) {
	case "MD5":
		return gosnmp.MD5
	case "SHA":
		return gosnmp.SHA
	case "SHA224":
		return gosnmp.SHA224
	case "SHA256":
		return gosnmp.SHA256
	case "SHA384":
		return gosnmp.SHA384
	case "SHA512":
		return gosnmp.SHA512
	default:
		return gosnmp.NoAuth
	}
}

func (a Auth) GetPrivProtocol() gosnmp.SnmpV3PrivProtocol {
	switch strings.ToUpper(a.PrivType) {
	case "DES":
		return gosnmp.DES
	case "AES":
		return gosnmp.AES
	case "AES192":
		return gosnmp.AES192
	case "AES192C":
		return gosnmp.AES192C
	case "AES256":
		return gosnmp.AES256
	case "AES256C":
		return gosnmp.AES256C
	default:
		return gosnmp.NoPriv
	}
}
