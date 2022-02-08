package push

import (
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/evcc-io/evcc/util"
)

// Event is a notification event
type Event struct {
	LoadPoint *int // optional loadpoint id
	Event     string
}

// EventTemplate is the push message template for an event
type EventTemplate struct {
	Title, Msg string
}

// Hub subscribes to event notifications and sends them to client devices
type Hub struct {
	definitions map[string]EventTemplate
	sender      []Sender
	cache       *util.Cache
}

// NewHub creates push hub with definitions and receiver
func NewHub(definitions map[string]EventTemplate, cache *util.Cache) *Hub {
	h := &Hub{
		definitions: definitions,
		cache:       cache,
	}
	return h
}

// Add adds a sender to the list of senders
func (h *Hub) Add(sender Sender) {
	h.sender = append(h.sender, sender)
}

// apply applies the event template to the content to produce the actual message
func (h *Hub) apply(ev Event, tmpl string) (string, error) {
	attr := make(map[string]interface{})

	// let cache catch up, refs reverted https://github.com/evcc-io/evcc/pull/445
	time.Sleep(100 * time.Millisecond)

	// get all values from cache
	for _, p := range h.cache.All() {
		if p.LoadPoint == nil || ev.LoadPoint == p.LoadPoint {
			attr[p.Key] = p.Val
		}
	}

	// init golang text template
	t, err := template.New("out").Funcs(template.FuncMap(sprig.FuncMap())).Parse(tmpl)
	if err != nil {
		return tmpl, err
	}

	// apply data attributes to template using sprig functions
	applied := new(strings.Builder)
	if err := t.Execute(applied, attr); err != nil {
		return "", err
	}

	return util.ReplaceFormatted(applied.String(), attr)
}

// Run is the Hub's main publishing loop
func (h *Hub) Run(events <-chan Event) {
	for ev := range events {
		if len(h.sender) == 0 {
			continue
		}

		definition, ok := h.definitions[ev.Event]
		if !ok {
			continue
		}

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
			go sender.Send(title, msg)
		}
	}
}
