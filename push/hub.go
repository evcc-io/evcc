package push

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/evcc-io/evcc/util"
)

// Event is a notification event
type Event struct {
	Loadpoint *int // optional loadpoint id
	Event     string
}

// EventTemplateConfig is the push message configuration for an event
type EventTemplateConfig struct {
	Title, Msg string
}

type Vehicles interface {
	// ByName returns a single vehicle adapter by name
	ByName(string) (vehicle.API, error)
}

// Hub subscribes to event notifications and sends them to client devices
type Hub struct {
	definitions map[string]EventTemplateConfig
	sender      []Messenger
	cache       *util.Cache
	vehicles    Vehicles
}

// NewHub creates push hub with definitions and receiver
func NewHub(cc map[string]EventTemplateConfig, vv Vehicles, cache *util.Cache) (*Hub, error) {
	// instantiate all event templates
	for k, v := range cc {
		if _, err := template.New("out").Funcs(sprig.TxtFuncMap()).Parse(v.Title); err != nil {
			return nil, fmt.Errorf("invalid event title: %s (%w)", k, err)
		}
		if _, err := template.New("out").Funcs(sprig.TxtFuncMap()).Parse(v.Msg); err != nil {
			return nil, fmt.Errorf("invalid event message: %s (%w)", k, err)
		}
	}

	h := &Hub{
		definitions: cc,
		cache:       cache,
		vehicles:    vv,
	}

	return h, nil
}

// Add adds a sender to the list of senders
func (h *Hub) Add(sender Messenger) {
	h.sender = append(h.sender, sender)
}

// apply applies the event template to the content to produce the actual message
func (h *Hub) apply(ev Event, tmpl string) (string, error) {
	attr := make(map[string]interface{})

	// loadpoint id
	if ev.Loadpoint != nil {
		attr["loadpoint"] = *ev.Loadpoint + 1
	}

	// get all values from cache
	for _, p := range h.cache.All() {
		if p.Loadpoint == nil || ev.Loadpoint == p.Loadpoint {
			attr[p.Key] = p.Val
		}
	}

	// add missing attributes
	if name, ok := attr["vehicleName"].(string); ok {
		if v, err := h.vehicles.ByName(name); err == nil {
			attr["vehicleTitle"] = v.Instance().Title()
		}
	}

	return util.ReplaceFormatted(tmpl, attr)
}

// Run is the Hub's main publishing loop
func (h *Hub) Run(events <-chan Event, valueChan chan util.Param) {
	log := util.NewLogger("push")

	for ev := range events {
		if len(h.sender) == 0 {
			continue
		}

		definition, ok := h.definitions[ev.Event]
		if !ok {
			continue
		}

		// let cache catch up, refs https://github.com/evcc-io/evcc/pull/445
		flushC := util.Flusher()
		valueChan <- util.Param{Val: flushC}
		<-flushC

		title, err := h.apply(ev, definition.Title)
		if err != nil {
			log.ERROR.Printf("invalid title template for %s: %v", ev.Event, err)
			continue
		}

		msg, err := h.apply(ev, definition.Msg)
		if err != nil {
			log.ERROR.Printf("invalid message template for %s: %v", ev.Event, err)
			continue
		}

		for _, sender := range h.sender {
			if strings.TrimSpace(msg) != "" {
				go sender.Send(title, msg)
			} else {
				log.DEBUG.Printf("did not send empty message template for %s: %v", ev.Event, err)
			}
		}
	}
}
