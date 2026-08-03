package vehicle

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type Publisher struct {
	Ch   chan<- util.Param
}

func NewPublisher(ch chan<- util.Param) *Publisher {
	return &Publisher{Ch: ch}
}

func (p *Publisher) Publish(key string, val any) {
	p.Ch <- util.Param{Key: key, Val: val}
}

func (p *Publisher) PublishAllVehicleData(vSettings []API,  v api.Vehicle) {
	id := ""
	for _, s := range vSettings {
		// TODO: does not work when title gets changed, need to find a better way to identify the vehicle
		if v.GetTitle() == s.Instance().GetTitle() {
			id = s.Name()
			break
		}
	}

	if id == "" {
		return
	}

	prefix := "vehicles." + id + "."

	// Battery SOC
	if soc, err := v.Soc(); err == nil {
		p.Publish(prefix+"soc", soc)
	}

	// Battery Capacity
	if cap := v.Capacity(); cap > 0 {
		p.Publish(prefix+"capacity", cap)
	}

	// Range
	if r, ok := api.Cap[api.VehicleRange](v); ok {
		if val, err := r.Range(); err == nil {
			p.Publish(prefix+"range", val)
		}
	}

	// Odometer
	if o, ok := api.Cap[api.VehicleOdometer](v); ok {
		if val, err := o.Odometer(); err == nil {
			p.Publish(prefix+"odometer", val)
		}
	}

	// Finish Timer
	if ft, ok := api.Cap[api.VehicleFinishTimer](v); ok {
		if val, err := ft.FinishTime(); err == nil {
			p.Publish(prefix+"finishTime", val)
		}
	}

	// Climater
	if cl, ok := api.Cap[api.VehicleClimater](v); ok {
		if val, err := cl.Climater(); err == nil {
			p.Publish(prefix+"climater", val)
		}
	}

	// Position
	if pos, ok := api.Cap[api.VehiclePosition](v); ok {
		if lat, lon, err := pos.Position(); err == nil {
			p.Publish(prefix+"latitude", lat)
			p.Publish(prefix+"longitude", lon)
		}
	}
}
