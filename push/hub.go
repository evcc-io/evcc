package push

// Hub subscribes to event notifications and sends them to client devices
type Hub struct {
	definitions map[string]EventTemplate
	sender      []Sender
}

// NewHub creates push hub with definitions and receiver
func NewHub(definitions map[string]EventTemplate) *Hub {
	h := &Hub{
		definitions: definitions,
	}
	return h
}

// Add adds a sender to the list of senders
func (h *Hub) Add(sender Sender) {
	h.sender = append(h.sender, sender)
}

// Run is the Hub's main publishing loop
func (h *Hub) Run(events <-chan Event) {
	for ev := range events {
		if len(h.sender) == 0 {
			continue
		}

		definition, ok := h.definitions[ev.Event]
		if !ok {
			log.ERROR.Printf("invalid event %v", ev.Event)
			break
		}

		msg, err := ev.apply(definition.Msg)
		if err != nil {
			log.ERROR.Printf("invalid message template: %v", err)
		}

		for _, sender := range h.sender {
			go sender.Send(ev, definition.Title, msg)
		}
	}
}
