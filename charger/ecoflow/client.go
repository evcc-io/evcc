// Ported from shuette42/ecoflow-energy-ha (MIT), see the package documentation.

package ecoflow

import (
	"cmp"
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/google/uuid"
)

const brokerURL = "wss://mqtt-e.ecoflow.com:8084/mqtt"

// probeAfter is the heartbeat gap after which a wallbox gets a get-all.
// Heartbeats normally arrive every 60 s.
const probeAfter = 90 * time.Second

// clientID builds a web app client ID WEB_{uuid}_{userId}_{appKey}_{ms}_{MD5}.
func clientID(userID string, app AppKey) string {
	base := "WEB_" + uuid.NewString() + "_" + userID
	ts := fmt.Sprint(time.Now().UnixMilli())
	sum := md5.Sum([]byte(app.Secret + base + ts))
	return base + "_" + app.Key + "_" + ts + "_" + strings.ToUpper(fmt.Sprintf("%x", sum))
}

// Connection is the app MQTT session of an EcoFlow account.
type Connection struct {
	log    *util.Logger
	client paho.Client
	userID string
	seq    atomic.Uint64

	mu        sync.Mutex
	wallboxes map[string]*Wallbox
}

type sharedConn struct {
	conn     *Connection
	password string
}

var (
	mu          sync.Mutex
	connections = make(map[string]*sharedConn)
)

// NewConnection returns the connection of the account, logging in and
// connecting on first use. All devices of the account share it, because
// EcoFlow drops concurrent sessions of one account.
func NewConnection(user, password string) (*Connection, error) {
	mu.Lock()
	defer mu.Unlock()

	if e, ok := connections[user]; ok {
		if e.password == password {
			return e.conn, nil
		}
		e.conn.client.Disconnect(250)
		delete(connections, user)
	}

	conn, err := connect(user, password)
	if err != nil {
		return nil, err
	}

	connections[user] = &sharedConn{conn: conn, password: password}
	return conn, nil
}

func connect(user, password string) (*Connection, error) {
	log := util.NewLogger("ecoflow").Redact(user, password, base64.StdEncoding.EncodeToString([]byte(password)))
	redactSession := log.RotatingSlot()

	apps, err := AppKeys()
	if err != nil {
		return nil, err
	}

	session, err := Login(log, redactSession, user, password)
	if err != nil {
		return nil, err
	}

	c := &Connection{
		log:       log,
		userID:    session.UserID,
		wallboxes: make(map[string]*Wallbox),
	}

	setSession := func(o *paho.ClientOptions) {
		o.SetClientID(clientID(session.UserID, apps[rand.IntN(len(apps))]))
		o.SetUsername(session.Account)
		o.SetPassword(session.Password)
	}

	opts := paho.NewClientOptions().
		AddBroker(brokerURL).
		SetCleanSession(true).
		SetKeepAlive(time.Minute).
		SetConnectTimeout(request.Timeout).
		SetMaxReconnectInterval(time.Minute).
		SetOnConnectHandler(c.onConnect).
		SetConnectionLostHandler(func(_ paho.Client, err error) {
			log.WARN.Printf("connection lost: %v", err)
		}).
		// the broker rejects a reused client ID and may have expired the credentials
		SetReconnectingHandler(func(_ paho.Client, o *paho.ClientOptions) {
			if s, err := Login(log, redactSession, user, password); err != nil {
				log.ERROR.Printf("login: %v", err)
			} else {
				session = s
			}
			setSession(o)
		})
	setSession(opts)

	c.client = paho.NewClient(opts)
	if token := c.client.Connect(); !token.WaitTimeout(request.Timeout) || token.Error() != nil {
		// a timed out connect would finish in the background and hold a second session
		c.client.Disconnect(0)
		return nil, fmt.Errorf("connect: %w", cmp.Or(token.Error(), api.ErrTimeout))
	}

	return c, nil
}

// onConnect restores subscriptions and snapshots after each (re)connect.
func (c *Connection) onConnect(paho.Client) {
	c.log.DEBUG.Printf("connected to %s", brokerURL)

	c.mu.Lock()
	wallboxes := slices.Collect(maps.Values(c.wallboxes))
	c.mu.Unlock()

	for _, wb := range wallboxes {
		if err := wb.subscribe(); err != nil {
			c.log.ERROR.Printf("%s: %v", wb.serial, err)
		}
	}
}

func (c *Connection) publish(topic string, payload []byte) error {
	return wait(c.client.Publish(topic, 1, false, payload))
}

func wait(token paho.Token) error {
	if !token.WaitTimeout(request.Timeout) {
		return api.ErrTimeout
	}
	return token.Error()
}

// Wallbox is a PowerPulse reached through a Connection.
type Wallbox struct {
	log       *util.Logger
	conn      *Connection
	serial    string
	gateway   string // PowerOcean that settings and start/stop go through
	heartbeat *util.Monitor[Heartbeat]

	mu   sync.Mutex
	seen time.Time // last heartbeat or get-all
}

// Wallbox subscribes to the telemetry of the PowerPulse until ctx is done.
// Telemetry older than timeout is outdated.
func (c *Connection) Wallbox(ctx context.Context, serial, gateway string, timeout time.Duration) (*Wallbox, error) {
	wb := &Wallbox{
		log:       c.log,
		conn:      c,
		serial:    serial,
		gateway:   gateway,
		heartbeat: util.NewMonitor[Heartbeat](timeout),
	}

	c.mu.Lock()
	c.wallboxes[serial] = wb
	c.mu.Unlock()

	if err := wb.subscribe(); err != nil {
		return nil, err
	}

	go wb.watch(ctx)

	return wb, nil
}

func (wb *Wallbox) subscribe() error {
	if err := wait(wb.conn.client.Subscribe("/app/device/property/"+wb.serial, 0, func(_ paho.Client, msg paho.Message) {
		wb.handle(msg.Payload())
	})); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	return wb.requestAll()
}

// watch requests a snapshot while heartbeats are missing. They regularly stop
// for minutes while other frames keep arriving, and a get-all brings them back.
func (wb *Wallbox) watch(ctx context.Context) {
	for tick := time.Tick(time.Minute); ; {
		select {
		case <-ctx.Done():
			wb.conn.mu.Lock()
			if wb.conn.wallboxes[wb.serial] == wb {
				delete(wb.conn.wallboxes, wb.serial)
			}
			wb.conn.mu.Unlock()
			return
		case <-tick:
		}

		wb.mu.Lock()
		silent := time.Since(wb.seen)
		wb.mu.Unlock()

		if silent >= probeAfter {
			wb.log.DEBUG.Printf("%s: no heartbeat since %v, requesting snapshot", wb.serial, silent.Round(time.Second))
			if err := wb.requestAll(); err != nil {
				wb.log.ERROR.Printf("%s: get-all: %v", wb.serial, err)
			}
		}
	}
}

// requestAll asks the wallbox for a full snapshot. It only pushes on change otherwise.
func (wb *Wallbox) requestAll() error {
	wb.mu.Lock()
	wb.seen = time.Now()
	wb.mu.Unlock()

	topic := fmt.Sprintf("/app/%s/%s/thing/property/get", wb.conn.userID, wb.serial)
	return wb.conn.publish(topic, BuildGetAll(wb.conn.seq.Add(1)))
}

// handle stores the heartbeats of a telemetry payload and ignores everything else.
func (wb *Wallbox) handle(payload []byte) {
	wb.log.TRACE.Printf("%s: recv %x", wb.serial, payload)

	headers, err := ParseMessage(payload)
	if err != nil {
		wb.log.TRACE.Printf("%s: undecodable payload: %v", wb.serial, err)
		return
	}

	for _, h := range headers {
		if fn, id, ok := h.Command(); !ok || fn != 2 || id != 33 {
			continue
		}

		for _, pdata := range h.Pdata() {
			hb, err := DecodeHeartbeat(pdata)
			if err != nil {
				wb.log.DEBUG.Printf("%s: heartbeat: %v", wb.serial, err)
				continue
			}
			wb.log.TRACE.Printf("%s: heartbeat %s", wb.serial, hb)

			hb.Received = time.Now()

			wb.mu.Lock()
			wb.seen = hb.Received
			wb.mu.Unlock()

			wb.heartbeat.SetFunc(func(prev Heartbeat) Heartbeat {
				// some heartbeats carry no energy counter, keep the last one
				if hb.TotalEnergyWh == 0 {
					hb.TotalEnergyWh = prev.TotalEnergyWh
				}
				return hb
			})
		}
	}
}

// Heartbeat returns the latest heartbeat, or api.ErrOutdated when none
// arrived within the timeout.
func (wb *Wallbox) Heartbeat() (Heartbeat, error) {
	return wb.heartbeat.Get()
}

// write publishes a command to the gateway or the wallbox itself. The wallbox
// sends no reply, the effect shows in later heartbeats.
func (wb *Wallbox) write(name, target string, build func(seq uint64) []byte) error {
	frame := build(wb.conn.seq.Add(1))
	wb.log.DEBUG.Printf("%s: send %s", wb.serial, name)
	wb.log.TRACE.Printf("%s: send %x", wb.serial, frame)

	topic := fmt.Sprintf("/app/%s/%s/thing/property/set", wb.conn.userID, target)
	if err := wb.conn.publish(topic, frame); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// StartStop starts or stops charging (241/100 via the gateway).
func (wb *Wallbox) StartStop(start bool) error {
	var selector uint64 = 1
	if start {
		selector = 2
	}

	return wb.write(fmt.Sprintf("enable=%t", start), wb.gateway, func(seq uint64) []byte {
		return BuildStartStop(wb.serial, selector, seq)
	})
}

// SetCurrent writes the session current in 0.1 A (ChargerCtrl 2/81 to the
// wallbox, heartbeat field 17). It switches the wallbox to its custom work mode.
func (wb *Wallbox) SetCurrent(raw uint64) error {
	return wb.write(fmt.Sprintf("current=%.1fA", float64(raw)/10), wb.serial, func(seq uint64) []byte {
		return BuildChargeCurrent(wb.serial, raw, seq)
	})
}

// SetPhaseMode writes PhaseOne or PhaseThree (241/102 via the gateway).
func (wb *Wallbox) SetPhaseMode(mode uint64) error {
	return wb.write(fmt.Sprintf("phasemode=%d", mode), wb.gateway, func(seq uint64) []byte {
		return BuildPhaseSet(wb.serial, wb.gateway, mode, seq)
	})
}
