package push

import (
	"github.com/andig/evcc/util"
)

// Event is a notification event
type Event struct {
	LoadPoint *int // optional loadpoint id
	Event     string
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
func (h *Hub) apply(ev Event, template string) (string, error) {
	attr := make(map[string]interface{})

	// get all values from cache
	for _, p := range h.cache.All() {
		if p.LoadPoint == nil || ev.LoadPoint == p.LoadPoint {
			attr[p.Key] = p.Val
		}
	}

	return util.ReplaceFormatted(template, attr)
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

		msg, err := h.apply(ev, definition.Msg)
		if err != nil {
			log.ERROR.Printf("invalid template for %s: %v", ev.Event, err)
			continue
		}

		for _, sender := range h.sender {
			go sender.Send(ev, definition.Title, msg)
		}
	}
}
