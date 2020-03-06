package cmd

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/core"
	"github.com/andig/evcc/core/wrapper"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/push"
	"github.com/spf13/viper"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// MQTT singleton
var mq *provider.MqttClient

func clientID() string {
	pid := rand.Int31()
	return fmt.Sprintf("evcc-%d", pid)
}

func configureMeters(conf config) (meters map[string]api.Meter) {
	meters = make(map[string]api.Meter)
	for _, mc := range conf.Meters {
		m := core.NewMeter(
			floatGetter(mc.Power),
		)

		if mc.Energy != nil {
			m = &wrapper.CompositeMeter{
				Meter:       m,
				MeterEnergy: core.NewMeterEnergy(floatGetter(mc.Energy)),
			}
		}
		meters[mc.Name] = m
	}
	return
}

func configureChargers(conf config) (chargers map[string]api.Charger) {
	chargers = make(map[string]api.Charger)
	for _, cc := range conf.Chargers {
		var c api.Charger

		switch cc.Type {
		case "wallbe":
			c = core.NewWallbe(cc.URI)

		case "default", "configurable":
			c = core.NewCharger(
				stringGetter(cc.Status),
				intProvider(cc.ActualCurrent),
				boolGetter(cc.Enabled),
				boolSetter("enable", cc.Enable),
			)

			// if chargecontroller specified build composite charger
			if cc.MaxCurrent != nil {
				c = &wrapper.CompositeCharger{
					Charger: c,
					ChargeController: core.NewChargeController(
						intSetter("current", cc.MaxCurrent),
					),
				}
			}
		default:
			log.FATAL.Fatalf("invalid charger type '%s'", cc.Type)
		}

		chargers[cc.Name] = c
	}
	return
}

func configureSoCs(conf config) (socs map[string]api.SoC) {
	socs = make(map[string]api.SoC)
	for _, sc := range conf.SoCs {
		soc := core.NewSoC(
			sc.Capacity,
			sc.Title,
			floatGetter(sc.Charge),
		)

		socs[sc.Name] = soc
	}
	return
}

func configureLoadPoint(lp *core.LoadPoint, lpc loadPointConfig) {
	if lpc.Mode != "" {
		// workaround for golangs yaml off=0 conversion
		if lpc.Mode == "0" {
			lpc.Mode = api.ModeOff
		}
		lp.Synced().SetMode(lpc.Mode)
	}
	if lpc.MinCurrent > 0 {
		lp.MinCurrent = lpc.MinCurrent
	}
	if lpc.MaxCurrent > 0 {
		lp.MaxCurrent = lpc.MaxCurrent
	}
	if lpc.Voltage > 0 {
		lp.Voltage = lpc.Voltage
	}
	if lpc.Phases > 0 {
		lp.Phases = lpc.Phases
	}
	if lpc.ResidualPower != 0 {
		lp.ResidualPower = lpc.ResidualPower
	}
}

func loadConfig(conf config, eventsChan chan push.Event) (loadPoints []*core.LoadPoint) {
	if viper.Get("mqtt") != nil {
		mq = provider.NewMqttClient(conf.Mqtt.Broker, conf.Mqtt.User, conf.Mqtt.Password, clientID(), 1)
	}

	meters := configureMeters(conf)
	chargers := configureChargers(conf)
	socs := configureSoCs(conf)

	for _, lpc := range conf.LoadPoints {
		charger, ok := chargers[lpc.Charger]
		if !ok {
			log.FATAL.Fatalf("invalid charger '%s'", lpc.Charger)
		}
		lp := core.NewLoadPoint(
			lpc.Name,
			charger,
		)

		// assign meters
		for _, m := range []struct {
			key   string
			meter *api.Meter
		}{
			{lpc.GridMeter, &lp.GridMeter},
			{lpc.ChargeMeter, &lp.ChargeMeter},
			{lpc.PVMeter, &lp.PVMeter},
		} {
			if m.key != "" {
				if impl, ok := meters[m.key]; ok {
					*m.meter = impl
				} else {
					log.FATAL.Fatalf("invalid meter '%s'", m.key)
				}
			}
		}

		// assign socs
		if lpc.SoC != "" {
			if impl, ok := socs[lpc.SoC]; ok {
				lp.SoC = impl
			} else {
				log.FATAL.Fatalf("invalid soc '%s'", lpc.SoC)
			}
		}

		// assign remaing config
		configureLoadPoint(lp, lpc)

		loadPoints = append(loadPoints, lp)
	}

	return
}
