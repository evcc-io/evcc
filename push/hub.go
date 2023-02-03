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
	Loadpoint *int // optional loadpoint id
	Event     string
}

// EventTemplateConfig is the push message configuration for an event
type EventTemplateConfig struct {
	Title, Msg string
}

// EventTemplate is the push message template for an event
type EventTemplate struct {
	Title, Msg *template.Template
}

// Hub subscribes to event notifications and sends them to client devices
type Hub struct {
	definitions map[string]EventTemplate
	sender      []Messenger
	cache       *util.Cache
}

// NewHub creates push hub with definitions and receiver
func NewHub(cc map[string]EventTemplateConfig, cache *util.Cache) (*Hub, error) {
	definitions := make(map[string]EventTemplate)

	// instantiate all event templates
	for k, v := range cc {
		var def EventTemplate
		var err error

		def.Title, err = template.New("out").Funcs(template.FuncMap(sprig.FuncMap())).Parse(v.Title)
		if err == nil {
			def.Msg, err = template.New("out").Funcs(template.FuncMap(sprig.FuncMap())).Parse(v.Msg)
		}

		if err != nil {
			return nil, err
		}

		definitions[k] = def
	}

	h := &Hub{
		definitions: definitions,
		cache:       cache,
	}

	return h, nil
}

// Add adds a sender to the list of senders
func (h *Hub) Add(sender Messenger) {
	h.sender = append(h.sender, sender)
}

// apply applies the event template to the content to produce the actual message
func (h *Hub) apply(ev Event, tmpl *template.Template) (string, error) {
	attr := make(map[string]interface{})

	// let cache catch up, refs reverted https://github.com/evcc-io/evcc/pull/445
	time.Sleep(100 * time.Millisecond)

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

	// apply data attributes to template using sprig functions
	applied := new(strings.Builder)
	if err := tmpl.Execute(applied, attr); err != nil {
		return "", err
	}

	return util.ReplaceFormatted(applied.String(), attr)
}

// Run is the Hub's main publishing loop
func (h *Hub) Run(events <-chan Event) {
	log := util.NewLogger("push")

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
			if strings.TrimSpace(msg) != "" {
				go sender.Send(title, msg)
			} else {
				log.DEBUG.Printf("did not send empty message template for %s: %v", ev.Event, err)
			}
		}
	}
}
