package tariffs

import (
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db/settings"
	"golang.org/x/text/currency"
)

const (
	SettingsKey = "tariffs."

	Grid    = "grid"
	Feedin  = "feedin"
	Planner = "planner"
	Co2     = "co2"
)

// Tariffs is the tariffs container
type Tariffs struct {
	mu                         sync.RWMutex
	currency                   currency.Unit
	grid, feedin, co2, planner api.Tariff
}

var _ API = (*Tariffs)(nil)

// New creates a new tariffs container
func New(currency currency.Unit) *Tariffs {
	return &Tariffs{
		currency: currency,
	}
}

func currentPrice(t api.Tariff) (float64, error) {
	if t != nil {
		if rr, err := t.Rates(); err == nil {
			if r, err := rr.Current(time.Now()); err == nil {
				return r.Price, nil
			}
		}
	}
	return 0, api.ErrNotAvailable
}

func (t *Tariffs) GetCurrency() currency.Unit {
	return t.currency
}

func (t *Tariffs) SetCurrency(s string) error {
	c, err := currency.ParseISO(s)
	if err == nil {
		t.currency = c
		settings.SetString(SettingsKey+"currency", s)
	}
	return err
}

func getRef(ref string, t api.Tariff) string {
	val, err := settings.String(SettingsKey + ref)
	if err == nil && val != "" {
		return val
	}
	if t != nil {
		return ref
	}
	return ""
}

func (t *Tariffs) GetRef(ref string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	switch ref {
	case Grid:
		return getRef(Grid, t.grid)
	case Feedin:
		return getRef(Feedin, t.feedin)
	case Planner:
		return getRef(Planner, t.planner)
	case Co2:
		return getRef(Co2, t.co2)
	default:
		panic("invalid tariff ref: " + ref)
	}
}

func (t *Tariffs) SetRef(ref, value string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	settings.SetString(SettingsKey+ref, value)
}

func (t *Tariffs) GetInstance(ref string) api.Tariff {
	t.mu.RLock()
	defer t.mu.RUnlock()

	switch ref {
	case Grid:
		return t.grid
	case Feedin:
		return t.feedin
	case Planner:
		return t.planner
	case Co2:
		return t.co2
	default:
		panic("invalid tariff ref: " + ref)
	}
}

func (t *Tariffs) SetInstance(ref string, tariff api.Tariff) {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch ref {
	case Grid:
		t.grid = tariff
	case Feedin:
		t.feedin = tariff
	case Planner:
		t.planner = tariff
	case Co2:
		t.co2 = tariff
	default:
		panic("invalid tariff ref: " + ref)
	}
}

// CurrentGridPrice returns the current grid price.
func (t *Tariffs) CurrentGridPrice() (float64, error) {
	return currentPrice(t.grid)
}

// CurrentFeedInPrice returns the current feed-in price.
func (t *Tariffs) CurrentFeedInPrice() (float64, error) {
	return currentPrice(t.feedin)
}

// CurrentCo2 determines the grids co2 emission.
func (t *Tariffs) CurrentCo2() (float64, error) {
	if t.co2 != nil {
		return currentPrice(t.co2)
	}
	return 0, api.ErrNotAvailable
}
