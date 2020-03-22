package push

// EventTemplate is the push message template for an event
type EventTemplate struct {
	title, msg string
}

// Hub subscribes to event notifications and sends them to client devices
type Hub struct {
	pushOver    *PushOver
	definitions map[string]EventTemplate
}

// NewHub creates push hub with definitions and receiver
func NewHub(definitions map[string]EventTemplate, pushOver *PushOver) *Hub {
	h := &Hub{
		pushOver:    pushOver,
		definitions: definitions,
	}
	return h
}

// Run is the Hub's main publishing loop
func (h *Hub) Run(events <-chan Event) {
	for ev := range events {
		if h.pushOver == nil {
			continue
		}

		definition, ok := h.definitions[ev.Event]
		if !ok {
			log.ERROR.Printf("invalid event %v", ev.Event)
			break
		}

		go h.pushOver.Send(ev, definition.title, definition.msg)
	}
}
