package snmp

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"
)

type Handler interface {
	Get(oids []string) (result *gosnmp.SnmpPacket, err error)
}

type Connection struct {
	sync.Mutex
	Handler Handler
}

func (c *Connection) Get(oids []string) (*gosnmp.SnmpPacket, error) {
	c.Lock()
	defer c.Unlock()
	return c.Handler.Get(oids)
}

func (c *Connection) Close() error {
	if cl, ok := c.Handler.(*gosnmp.GoSNMP); ok && cl.Conn != nil {
		return cl.Conn.Close()
	}
	return nil
}

var (
	mu          sync.Mutex
	connections = make(map[string]*Connection)
)

func parseTarget(uri string) (host string, port uint16, scheme string, err error) {
	if !strings.Contains(uri, "://") {
		uri = "udp://" + uri
	}

	u, err := url.Parse(uri)
	if err != nil {
		return "", 0, "", err
	}

	host = u.Hostname()
	portStr := u.Port()
	if portStr == "" {
		portStr = "161"
	}
	p, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return "", 0, "", fmt.Errorf("invalid port: %w", err)
	}

	return host, uint16(p), u.Scheme, nil
}

func buildCacheKey(host string, port uint16, version, community string, auth Auth) string {
	if strings.ToLower(version) != "3" {
		return fmt.Sprintf("%s:%d:%s:%s", host, port, version, community)
	}

	// For v3, include all security parameters in the key
	h := sha256.New()
	h.Write([]byte(auth.User))
	h.Write([]byte(auth.SecurityLevel))
	h.Write([]byte(auth.AuthType))
	h.Write([]byte(auth.AuthPassword))
	h.Write([]byte(auth.PrivType))
	h.Write([]byte(auth.PrivPassword))

	return fmt.Sprintf("%s:%d:%s:%x", host, port, version, h.Sum(nil))
}

func newGoSNMP(host string, port uint16, scheme, version, community string, auth Auth) (*gosnmp.GoSNMP, error) {
	g := &gosnmp.GoSNMP{
		Target:    host,
		Port:      port,
		Transport: scheme,
		Retries:   2,
		Timeout:   2 * time.Second,
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
	return g, nil
}

func NewConnection(ctx context.Context, uri, version, community string, auth Auth) (*Connection, error) {
	host, port, scheme, err := parseTarget(uri)
	if err != nil {
		return nil, err
	}

	key := buildCacheKey(host, port, version, community, auth)

	mu.Lock()
	if conn, ok := connections[key]; ok {
		mu.Unlock()
		return conn, nil
	}
	defer mu.Unlock()

	g, err := newGoSNMP(host, port, scheme, version, community, auth)
	if err != nil {
		return nil, err
	}

	res := &Connection{Handler: g}
	connections[key] = res

	if ctx != nil {
		go func() {
			<-ctx.Done()
			mu.Lock()
			delete(connections, key)
			mu.Unlock()
			_ = res.Close()
		}()
	}

	return res, nil
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
