package solarman

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"
	"syscall"

	"github.com/evcc-io/evcc/util"
)

type SolarmanConnection interface {
	reconnect() error
	writeToSocket(request []byte) error
	readFromSocket() ([]byte, error)
	Send(request []byte) ([]byte, error)
}

type ConnectionProvider interface {
	CreateConnection(string) (net.Conn, error)
}

type DefaultConnectionProvider struct {
}

func (dcp *DefaultConnectionProvider) CreateConnection(uri string) (net.Conn, error) {
	return net.Dial("tcp", uri)
}

type LSW3Connection struct {
	conn                net.Conn
	connection_provider ConnectionProvider
	uri                 string
	rw_mutex            sync.Mutex
	logger              *util.Logger
}

var (
	connections       map[string]*LSW3Connection = make(map[string]*LSW3Connection)
	connections_mutex sync.Mutex
)

func GetConnection(URI string) SolarmanConnection {
	connections_mutex.Lock()
	defer connections_mutex.Unlock()

	v, ok := connections[URI]
	if !ok {
		logger := util.NewLogger("solarman")
		logger.INFO.Printf("opening new connection to %s", URI)
		v = &LSW3Connection{
			connection_provider: &DefaultConnectionProvider{},
			logger:              logger,
			uri:                 URI,
		}
		conn, err := v.connection_provider.CreateConnection(URI)
		if err != nil {
			panic("could not open connection")
		}
		v.conn = conn
		connections[URI] = v
	}
	return v

}

func (c *LSW3Connection) reconnect() error {
	connections_mutex.Lock()
	defer connections_mutex.Unlock()
	c.conn.Close()
	conn, err := c.connection_provider.CreateConnection(c.uri)
	if err != nil {
		c.logger.ERROR.Printf("could not re-connect")
		return err
	}
	c.conn = conn
	return nil
}

func (c *LSW3Connection) writeToSocket(request []byte) error {
	n, err := c.conn.Write(request)
	if err != nil {
		return err
	}
	if n < len(request) {
		return fmt.Errorf("error while sending data (less data than expected)")
	}
	return nil
}

func (c *LSW3Connection) readFromSocket() ([]byte, error) {
	raw_response := make([]byte, 1024)
	n, err := c.conn.Read(raw_response)
	if err != nil {
		return nil, err
	}
	return raw_response[0:n], nil

}

func (c *LSW3Connection) Send(request []byte) ([]byte, error) {
	c.rw_mutex.Lock()
	defer c.rw_mutex.Unlock()

	err := c.writeToSocket(request)

	if errors.Is(err, syscall.EPIPE) {
		if err := c.reconnect(); err != nil {
			c.logger.INFO.Printf("could not re-connect")
			return nil, err
		}
		err = c.writeToSocket(request)
		if err != nil {
			return nil, err
		}
		c.logger.INFO.Printf("successfully re-connected and re-send data")

	}
	c.logger.TRACE.Printf("SENT %s", hex.EncodeToString(request))

	response, err := c.readFromSocket()
	if errors.Is(err, syscall.EPIPE) {
		if err := c.reconnect(); err != nil {
			c.logger.INFO.Printf("could not re-connect")
			return nil, err
		}
		c.logger.INFO.Printf("re-send data")
		err := c.writeToSocket(request)
		if err != nil {
			return nil, err
		}
		c.logger.INFO.Printf("re-read data")
		response, err = c.readFromSocket()
		if err != nil {
			return nil, err
		}
	}
	c.logger.TRACE.Printf("RECD %s", hex.EncodeToString(response))
	return response, nil
}
