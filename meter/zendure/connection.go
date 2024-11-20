package zendure

import (
	"encoding/json"
	"net"
	"strconv"
	"sync"
	"time"

	"dario.cat/mergo"
	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/util"
)

var (
	mu          sync.Mutex
	connections = make(map[string]*Connection)
)

type Connection struct {
	log  *util.Logger
	data *util.Monitor[Data]
}

func NewConnection(region, account, serial string, timeout time.Duration) (*Connection, error) {
	mu.Lock()
	defer mu.Unlock()

	key := account + serial
	if conn, ok := connections[key]; ok {
		return conn, nil
	}

	log := util.NewLogger("zendure")
	res, err := MqttCredentials(log, region, account, serial)
	if err != nil {
		return nil, err
	}

	client, err := mqtt.NewClient(
		log,
		net.JoinHostPort(res.Data.MqttUrl, strconv.Itoa(res.Data.Port)), res.Data.AppKey, res.Data.Secret,
		"", 0, false, "", "", "",
	)
	if err != nil {
		return nil, err
	}

	conn := &Connection{
		log:  log,
		data: util.NewMonitor[Data](timeout),
	}

	topic := res.Data.AppKey + "/#"
	if err := client.Listen(topic, conn.handler); err != nil {
		return nil, err
	}

	connections[key] = conn

	return conn, nil
}

func (c *Connection) handler(data string) {
	var res Payload
	if err := json.Unmarshal([]byte(data), &res); err != nil {
		c.log.ERROR.Println(err)
		return
	}

	if res.Data == nil {
		return
	}

	c.data.SetFunc(func(v Data) Data {
		if err := mergo.Merge(&v, res.Data); err != nil {
			c.log.ERROR.Println(err)
		}

		return v
	})
}

func (c *Connection) Data() (Data, error) {
	return c.data.Get()
}
