package vehicle

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type Publisher struct {
	Ch chan<- util.Param
}

func NewPublisher(ch chan<- util.Param) *Publisher {
	return &Publisher{Ch: ch}
}

func (p *Publisher) Publish(v string, key string, val any) {
	p.Ch <- util.Param{Vehicle: &v, Key: key, Val: val}
}

func (p *Publisher) PublishAllVehicleData(vSettings []API, v api.Vehicle) {
	id := findVehicleID(vSettings, v)
	if id == "" {
		return
	}

	// Battery SOC
	if soc, err := v.Soc(); err == nil {
		p.Publish(id, "soc", soc)
	}

	// Battery Capacity
	if cap := v.Capacity(); cap > 0 {
		p.Publish(id, "capacity", cap)
	}

	// Range
	if r, ok := api.Cap[api.VehicleRange](v); ok {
		p.publishIf(id, "range", func() (any, error) { return r.Range() })
	}

	// Odometer
	if o, ok := api.Cap[api.VehicleOdometer](v); ok {
		p.publishIf(id, "odometer", func() (any, error) { return o.Odometer() })
	}

	// Finish Timer
	if ft, ok := api.Cap[api.VehicleFinishTimer](v); ok {
		p.publishIf(id, "finishTime", func() (any, error) { return ft.FinishTime() })
	}

	// Climater
	if cl, ok := api.Cap[api.VehicleClimater](v); ok {
		p.publishIf(id, "climater", func() (any, error) { return cl.Climater() })
	}

	// Position
	if pos, ok := api.Cap[api.VehiclePosition](v); ok {
		if lat, lon, err := pos.Position(); err == nil {
			p.Publish(id, "latitude", lat)
			p.Publish(id, "longitude", lon)
		}
	}
}

func (p *Publisher) publishIf(v string, key string, get func() (any, error)) {
	if val, err := get(); err == nil {
		p.Publish(v, key, val)
	}
}

func findVehicleID(vSettings []API, v api.Vehicle) string {
	for _, s := range vSettings {
		if v.GetTitle() == s.Instance().GetTitle() {
			return s.Name()
		}
	}
	return ""
}
