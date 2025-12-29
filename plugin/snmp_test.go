package plugin

import (
	"testing"

	"github.com/evcc-io/evcc/util/snmp"
	"github.com/gosnmp/gosnmp"
	"github.com/stretchr/testify/assert"
)

type mockHandler struct {
	variables []gosnmp.SnmpPDU
}

func (m *mockHandler) Get(oids []string) (*gosnmp.SnmpPacket, error) {
	return &gosnmp.SnmpPacket{
		Variables: m.variables,
	}, nil
}

func TestSnmp(t *testing.T) {
	m := &mockHandler{
		variables: []gosnmp.SnmpPDU{
			{
				Name:  ".1.3.6.1.4.1.x.x.x",
				Type:  gosnmp.Integer,
				Value: 1234,
			},
		},
	}

	p := &Snmp{
		conn:  &snmp.Connection{Handler: m},
		oid:   ".1.3.6.1.4.1.x.x.x",
		scale: 0.1,
	}

	g, err := p.FloatGetter()
	assert.NoError(t, err)

	val, err := g()
	assert.NoError(t, err)
	assert.Equal(t, 123.4, val)
}

func TestSnmpOctetString(t *testing.T) {
	m := &mockHandler{
		variables: []gosnmp.SnmpPDU{
			{
				Name:  ".1.3.6.1.4.1.x.x.x",
				Type:  gosnmp.OctetString,
				Value: []byte("567.8"),
			},
		},
	}

	p := &Snmp{
		conn:  &snmp.Connection{Handler: m},
		oid:   ".1.3.6.1.4.1.x.x.x",
		scale: 1.0,
	}

	g, err := p.FloatGetter()
	assert.NoError(t, err)

	val, err := g()
	assert.NoError(t, err)
	assert.Equal(t, 567.8, val)
}

func TestSnmpEmptyOID(t *testing.T) {
	p := &Snmp{
		oid: "",
	}

	g, err := p.FloatGetter()
	assert.NoError(t, err)

	val, err := g()
	assert.NoError(t, err)
	assert.Equal(t, 0.0, val)
}
