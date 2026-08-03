package vehicle

import "github.com/evcc-io/evcc/util"
import "github.com/evcc-io/evcc/api"

type Publisher struct {
	Ch chan<- util.Param
}

func NewPublisher(ch chan<- util.Param) *Publisher {
	return &Publisher{Ch: ch}
}

func (p *Publisher) Publish(key string, val any) {
	p.Ch <- util.Param{Key: key, Val: val}
}

func PublishAllVehicleData(v api.Vehicle, pub *Publisher, id string) {
    prefix := "vehicles." + id + "."

    // Battery SOC
    if soc, err := v.Soc(); err == nil {
        pub.Publish(prefix+"soc", soc)
    }

    // Battery Capacity
    if cap := v.Capacity(); cap > 0 {
        pub.Publish(prefix+"capacity", cap)
    }

    // Range
    if r, ok := api.Cap[api.VehicleRange](v); ok {
        if val, err := r.Range(); err == nil {
            pub.Publish(prefix+"range", val)
        }
    }

    // Odometer
    if o, ok := api.Cap[api.VehicleOdometer](v); ok {
        if val, err := o.Odometer(); err == nil {
            pub.Publish(prefix+"odometer", val)
        }
    }

    // Finish Timer
    if ft, ok := api.Cap[api.VehicleFinishTimer](v); ok {
        if val, err := ft.FinishTime(); err == nil {
            pub.Publish(prefix+"finishTime", val)
        }
    }

    // Climater
    if cl, ok := api.Cap[api.VehicleClimater](v); ok {
        if val, err := cl.Climater(); err == nil {
            pub.Publish(prefix+"climater", val)
        }
    }

    // Position
    if pos, ok := api.Cap[api.VehiclePosition](v); ok {
        if lat, lon, err := pos.Position(); err == nil {
            pub.Publish(prefix+"latitude", lat)
            pub.Publish(prefix+"longitude", lon)
        }
    }
}

