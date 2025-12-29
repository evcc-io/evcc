package snmp

import (
	"context"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/stretchr/testify/assert"
)

type mockHandler struct {
}

func (m *mockHandler) Get(oids []string) (*gosnmp.SnmpPacket, error) {
	return &gosnmp.SnmpPacket{}, nil
}

func TestSharedConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host := "localhost"
	version := "2c"
	community := "public"
	auth := Auth{}

	key := buildCacheKey(host, 161, version, community, auth)

	conn := &Connection{Handler: &mockHandler{}}
	mu.Lock()
	connections[key] = conn
	mu.Unlock()
	defer func() {
		mu.Lock()
		delete(connections, key)
		mu.Unlock()
	}()

	res, err := NewConnection(ctx, host, version, community, auth)
	assert.NoError(t, err)
	assert.Equal(t, conn, res)
}

func TestUnregistration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	key := "unregister-test"
	conn := &Connection{Handler: &mockHandler{}}

	mu.Lock()
	connections[key] = conn
	mu.Unlock()

	go func() {
		<-ctx.Done()
		mu.Lock()
		delete(connections, key)
		mu.Unlock()
	}()

	cancel()
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	_, ok := connections[key]
	mu.Unlock()
	assert.False(t, ok)
}

func TestBuildCacheKey(t *testing.T) {
	host := "1.2.3.4"
	port := uint16(161)

	// v2c
	k1 := buildCacheKey(host, port, "2c", "public", Auth{})
	k2 := buildCacheKey(host, port, "2c", "public", Auth{})
	assert.Equal(t, k1, k2)
	assert.Equal(t, "1.2.3.4:161:2c:public", k1)

	// v3
	auth1 := Auth{User: "user", SecurityLevel: "authPriv", AuthPassword: "password"}
	auth2 := Auth{User: "user", SecurityLevel: "authPriv", AuthPassword: "password"}
	auth3 := Auth{User: "user", SecurityLevel: "authPriv", AuthPassword: "different"}

	kv3_1 := buildCacheKey(host, port, "3", "", auth1)
	kv3_2 := buildCacheKey(host, port, "3", "", auth2)
	kv3_3 := buildCacheKey(host, port, "3", "", auth3)

	assert.Equal(t, kv3_1, kv3_2)
	assert.NotEqual(t, kv3_1, kv3_3)
}
