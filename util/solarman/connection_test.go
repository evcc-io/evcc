package solarman

import (
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

type MockNetConn struct {
	data          []byte
	readCalled    int
	writeCalled   int
	readFunction  func(*MockNetConn, []byte) (int, error)
	writeFunction func(*MockNetConn, []byte) (int, error)
}

func (c *MockNetConn) Read(b []byte) (n int, err error) {
	n, err = c.readFunction(c, b)
	c.readCalled++
	return n, err
}

func (c *MockNetConn) Write(b []byte) (n int, err error) {
	n, err = c.writeFunction(c, b)
	c.writeCalled++
	return n, err
}

func (c *MockNetConn) Close() error {
	return nil
}

func (c *MockNetConn) LocalAddr() net.Addr {
	return nil
}

func (c *MockNetConn) RemoteAddr() net.Addr {
	return nil
}

func (c *MockNetConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *MockNetConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *MockNetConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (c *MockNetConn) Reset() {
	c.data = nil
}

type MockConnectionProvider struct {
	Count      int
	providerFn func(string) (net.Conn, error)
}

func (mcp *MockConnectionProvider) CreateConnection(uri string) (net.Conn, error) {
	mcp.Count++
	return mcp.providerFn(uri)
}

func TestSend(t *testing.T) {
	mockConnection := &MockNetConn{
		readFunction: func(mnc *MockNetConn, b []byte) (int, error) {
			copy(b[0:], mnc.data)
			return len(mnc.data), nil
		},
		writeFunction: func(mnc *MockNetConn, b []byte) (int, error) {
			mnc.data = b
			return len(mnc.data), nil
		},
	}
	mockConnectionProvider := &MockConnectionProvider{
		providerFn: func(s string) (net.Conn, error) {
			return mockConnection, nil
		},
	}

	connection := &LSW3Connection{
		logger:              util.NewLogger("solarman_test"),
		connection_provider: mockConnectionProvider,
		conn:                mockConnection,
	}

	response, err := connection.Send([]byte{0x01, 0x02})
	require.Equal(t, []byte{0x01, 0x02}, response)
	require.Equal(t, mockConnectionProvider.Count, 0)
	require.Nil(t, err)
}

func TestReconnectOnBrokenPipeRead(t *testing.T) {
	mockConnection := &MockNetConn{
		readFunction: func(mnc *MockNetConn, b []byte) (int, error) {
			if mnc.readCalled == 0 {
				return 0, syscall.EPIPE
			}
			copy(b[0:], mnc.data)
			return len(mnc.data), nil
		},
		writeFunction: func(mnc *MockNetConn, b []byte) (int, error) {
			mnc.data = b
			return len(mnc.data), nil
		},
	}
	mockConnectionProvider := &MockConnectionProvider{
		providerFn: func(s string) (net.Conn, error) {
			return mockConnection, nil
		},
	}

	connection := &LSW3Connection{
		logger:              util.NewLogger("solarman_test"),
		connection_provider: mockConnectionProvider,
		conn:                mockConnection,
	}

	response, err := connection.Send([]byte{0x01, 0x02})
	require.Equal(t, []byte{0x01, 0x02}, response)
	require.Nil(t, err)
	require.Equal(t, mockConnectionProvider.Count, 1)
}

func TestReconnectOnBrokenPipeWrite(t *testing.T) {
	mockConnection := &MockNetConn{
		readFunction: func(mnc *MockNetConn, b []byte) (int, error) {
			copy(b[0:], mnc.data)
			return len(mnc.data), nil
		},
		writeFunction: func(mnc *MockNetConn, b []byte) (int, error) {
			if mnc.writeCalled == 0 {
				return 0, syscall.EPIPE
			}
			mnc.data = b
			return len(mnc.data), nil
		},
	}
	mockConnectionProvider := &MockConnectionProvider{
		providerFn: func(s string) (net.Conn, error) {
			return mockConnection, nil
		},
	}

	connection := &LSW3Connection{
		logger:              util.NewLogger("solarman_test"),
		connection_provider: mockConnectionProvider,
		conn:                mockConnection,
	}

	response, err := connection.Send([]byte{0x01, 0x02})
	require.Equal(t, []byte{0x01, 0x02}, response)
	require.Nil(t, err)
	require.Equal(t, mockConnectionProvider.Count, 1)
}
