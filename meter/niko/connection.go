package niko

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin/mqtt"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

const (
	// the controller disables instant usage reporting 30s after it has been requested
	refreshInterval = 20 * time.Second
	maxAge          = 3 * refreshInterval
)

// Connection is a Niko Home Control smart plug connected via the Hobby API
type Connection struct {
	log    *util.Logger
	client *mqtt.Client
	uuid   string

	enableMu sync.Mutex

	mu      sync.Mutex
	status  bool
	power   float64
	updated time.Time // last report of the device

	once   sync.Once
	listed chan struct{}
	err    error
}

// Broker returns the Hobby API broker address
func Broker(host string) string {
	return "tls://" + util.DefaultPort(host, 8884)
}

// NewConnection creates a Niko Home Control smart plug connection
func NewConnection(ctx context.Context, host, user, token, uuid string) (*Connection, error) {
	switch {
	case token == "":
		return nil, api.ErrMissingToken
	case host == "" || uuid == "":
		return nil, errors.New("missing host or device")
	}

	user = cmp.Or(user, "hobby")
	log := util.NewLogger("niko").Redact(token)

	// the controller uses a self-signed certificate
	client, err := mqtt.RegisteredClient(log, Broker(host), user, token, "", 1, true, "", "", "")
	if err != nil {
		return nil, err
	}

	c := &Connection{
		log:    log,
		client: client,
		uuid:   uuid,
		listed: make(chan struct{}),
	}

	for _, topic := range []string{TopicRsp, TopicEvt} {
		if err := client.Listen(topic, c.handle); err != nil {
			return nil, err
		}
	}

	client.Publish(TopicCmd, false, ListRequest())

	select {
	case <-c.listed:
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(request.Timeout):
		return nil, fmt.Errorf("device list: %w", api.ErrTimeout)
	}

	if c.err != nil {
		return nil, c.err
	}

	c.refresh()
	go c.run(ctx)

	return c, nil
}

func (c *Connection) handle(payload string) {
	var msg Message
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		c.log.ERROR.Printf("invalid message: %v", err)
		return
	}

	found := c.update(msg)

	if msg.Method == MethodList {
		c.once.Do(func() {
			if !found {
				c.err = fmt.Errorf("device not found: %s", c.uuid)
			}
			close(c.listed)
		})
	}
}

// update applies the device's properties contained in msg and reports if the device was found
func (c *Connection) update(msg Message) bool {
	var found bool

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, d := range msg.Devices() {
		if d.Uuid != c.uuid {
			continue
		}
		found = true
		c.updated = time.Now()

		if status, ok := d.Status(); ok {
			c.status = status
		}
		if power, ok := d.Power(); ok {
			c.power = power
		}
	}

	return found
}

// refresh re-enables instant usage reporting and re-lists the devices, the latter
// restores the cached state after missed events, e.g. on reconnect
func (c *Connection) refresh() {
	c.client.Publish(TopicCmd, false, ControlRequest(c.uuid, "ReportInstantUsage", "True"))
	c.client.Publish(TopicCmd, false, ListRequest())
}

func (c *Connection) run(ctx context.Context) {
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.refresh()
		}
	}
}

// outdated must be called with the lock held
func (c *Connection) outdated() error {
	if time.Since(c.updated) > maxAge {
		return api.ErrOutdated
	}
	return nil
}

// Enabled returns the relay state
func (c *Connection) Enabled() (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status, c.outdated()
}

// Enable switches the relay and waits for the state change to be reported
func (c *Connection) Enable(enable bool) error {
	c.enableMu.Lock()
	defer c.enableMu.Unlock()

	onoff := map[bool]string{true: "On", false: "Off"}
	c.client.Publish(TopicCmd, false, ControlRequest(c.uuid, "Status", onoff[enable]))

	timeout := time.After(request.Timeout)
	for {
		if enabled, err := c.Enabled(); err == nil && enabled == enable {
			return nil
		}

		select {
		case <-timeout:
			return fmt.Errorf("switch %s: %w", onoff[enable], api.ErrTimeout)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

var _ api.Meter = (*Connection)(nil)

// CurrentPower implements the api.Meter interface
func (c *Connection) CurrentPower() (float64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.power, c.outdated()
}
