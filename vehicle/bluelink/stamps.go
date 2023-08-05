package bluelink

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

const (
	KiaAppID     = "a2b8469b-30a3-4361-8e13-6fceea8fbe74"
	HyundaiAppID = "014d2225-8495-4735-812d-2616334fd15d"
)

type stampCollection struct {
	mu           sync.Mutex
	log          *util.Logger
	AppID, Brand string
	Stamps       []string
	Generated    time.Time
	Frequency    int
	updated      time.Time
}

var (
	client = request.NewHelper(util.NewLogger("http"))

	Stamps = map[string]*stampCollection{
		KiaAppID:     {log: util.NewLogger("kia"), AppID: KiaAppID, Brand: "kia"},
		HyundaiAppID: {log: util.NewLogger("hyundai"), AppID: HyundaiAppID, Brand: "hyundai"},
	}
)

// New creates a new stamp
func (c *stampCollection) Get() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var position int
	if c.Frequency > 0 {
		position = int(time.Since(c.Generated).Milliseconds()) / c.Frequency
	}

	// download
	if position >= len(c.Stamps)*9/10 {
		if time.Since(c.updated) > 15*time.Minute {
			c.log.TRACE.Printf("retry stamps download, last attempt: %v", c.updated)
			if err := c.download(); err != nil {
				return "", err
			}
		}

		position = int(time.Since(c.Generated).Milliseconds()) / c.Frequency
	}

	if len(c.Stamps) == 0 {
		return "", errors.New("no stamps available")
	}

	if position >= len(c.Stamps) {
		position = len(c.Stamps) - 1
	}

	return c.Stamps[position], nil
}

// updateStamps updates stamps according to https://github.com/Hacksore/bluelinky/pull/144
func (c *stampCollection) download() error {
	c.updated = time.Now()

	uri := fmt.Sprintf("https://raw.githubusercontent.com/neoPix/bluelinky-stamps/master/%s-%s.v2.json", c.Brand, c.AppID)

	if err := client.GetJSON(uri, &c); err != nil {
		return fmt.Errorf("failed to download stamps: %w", err)
	}

	return nil
}
