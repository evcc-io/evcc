package push

var definitions = map[EventId]struct{ title, msg string }{
	ChargeStart: {
		title: "Charge started",
		msg:   "Loadpoint ${lp} started charging in \"${mode}\" mode",
	},
	ChargeStop: {
		title: "Charge finished",
		msg:   "Loadpoint ${lp} finished charging. Charged ${energy:%.1f}kWh in ${duration}.",
	},
}

// Hub subscribes to event notifications and sends them to client devices
type Hub struct {
	PushOver *PushOver
}

// Run is the Hub's main publishing loop
func (h *Hub) Run(events <-chan Event) {
	for ev := range events {
		if h.PushOver == nil {
			continue
		}

		definition, ok := definitions[ev.EventId]
		if !ok {
			log.ERROR.Printf("invalid event %v", ev.EventId)
			break
		}

		go h.PushOver.Send(ev, definition.title, definition.msg)
	}
}
