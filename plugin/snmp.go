package plugin

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/snmp"
	"github.com/gosnmp/gosnmp"
)

type Snmp struct {
	conn  *snmp.Connection
	oid   string
	scale float64
}

func init() {
	registry.AddCtx("snmp", NewSnmpFromConfig)
}

func NewSnmpFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	cc := struct {
		URI       string
		Version   string
		Community string
		Auth      snmp.Auth
		OID       string
		Scale     float64
	}{
		Version:   "2c",
		Community: "public",
		Scale:     1.0,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, fmt.Errorf("missing uri")
	}

	conn, err := snmp.NewConnection(ctx, cc.URI, cc.Version, cc.Community, cc.Auth)
	if err != nil {
		return nil, err
	}

	return &Snmp{
		conn:  conn,
		oid:   cc.OID,
		scale: cc.Scale,
	}, nil
}

func (p *Snmp) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		if p.oid == "" {
			return "0", nil
		}
		result, err := p.conn.Get([]string{p.oid})
		if err != nil {
			return "", fmt.Errorf("snmp get %s: %w", p.oid, err)
		}

		if len(result.Variables) == 0 {
			return "", fmt.Errorf("oid not found: %s", p.oid)
		}

		variable := result.Variables[0]
		switch variable.Type {
		case gosnmp.OctetString:
			return string(variable.Value.([]byte)), nil
		case gosnmp.Integer, gosnmp.Counter32, gosnmp.Gauge32, gosnmp.TimeTicks, gosnmp.Counter64:
			return strconv.FormatInt(gosnmp.ToBigInt(variable.Value).Int64(), 10), nil
		case gosnmp.Uinteger32:
			return strconv.FormatUint(uint64(variable.Value.(uint32)), 10), nil
		default:
			return fmt.Sprintf("%v", variable.Value), nil
		}
	}, nil
}

func (p *Snmp) FloatGetter() (func() (float64, error), error) {
	g, err := p.StringGetter()
	if err != nil {
		return nil, err
	}

	return func() (float64, error) {
		s, err := g()
		if err != nil {
			return 0, err
		}

		f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			return 0, err
		}

		return f * p.scale, nil
	}, nil
}

func (p *Snmp) IntGetter() (func() (int64, error), error) {
	g, err := p.FloatGetter()
	if err != nil {
		return nil, err
	}

	return func() (int64, error) {
		f, err := g()
		if err != nil {
			return 0, err
		}

		return int64(f), nil
	}, nil
}
