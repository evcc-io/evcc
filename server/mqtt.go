package server

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/util"
)

var deprecatedTopics = []string{
	"activePhases", "range", "socCharge",
	"vehicleSoC", "batterySoC", "bufferSoC", "minSoC", "prioritySoC", "targetSoC", "vehicleTargetSoC", "vehicles",
	"savingsAmount", "savingsEffectivePrice", "savingsGridCharged", "savingsSelfConsumptionCharged", "savingsSelfConsumptionPercent", "savingsTotalCharged",
}

// MQTT is the MQTT server. It uses the MQTT client for publishing.
type MQTT struct {
	log       *util.Logger
	Handler   *mqtt.Client
	root      string
	publisher func(topic string, retained bool, payload string)
}

// NewMQTT creates MQTT server
func NewMQTT(root string, site site.API) (*MQTT, error) {
	m := &MQTT{
		log:     util.NewLogger("mqtt"),
		Handler: mqtt.Instance,
		root:    root,
	}
	m.publisher = m.publishString

	err := m.Listen(site)
	return m, err
}

func (m *MQTT) encode(v interface{}) string {
	// nil should erase the value
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%.5g", val)
	case time.Time:
		if val.IsZero() {
			return ""
		}
		return strconv.FormatInt(val.Unix(), 10)
	case time.Duration:
		// must be before stringer to convert to seconds instead of string
		return fmt.Sprintf("%d", int64(val.Seconds()))
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprintf("%v", val)
	}
}

func (m *MQTT) publishComplex(topic string, retained bool, payload interface{}) {
	if _, ok := payload.(fmt.Stringer); ok || payload == nil {
		m.publishSingleValue(topic, retained, payload)
		return
	}

	switch typ := reflect.TypeOf(payload); typ.Kind() {
	case reflect.Slice:
		// publish count
		val := reflect.ValueOf(payload)
		m.publishSingleValue(topic, retained, val.Len())

		// loop slice
		for i := 0; i < val.Len(); i++ {
			m.publishComplex(fmt.Sprintf("%s/%d", topic, i+1), retained, val.Index(i).Interface())
		}

	case reflect.Map:
		// loop map
		for iter := reflect.ValueOf(payload).MapRange(); iter.Next(); {
			k := iter.Key().String()
			m.publishComplex(fmt.Sprintf("%s/%s", topic, k), retained, iter.Value().Interface())
		}

	case reflect.Struct:
		val := reflect.ValueOf(payload)
		typ := val.Type()

		// loop struct
		for i := 0; i < typ.NumField(); i++ {
			if f := typ.Field(i); f.IsExported() {
				n := f.Name
				m.publishComplex(fmt.Sprintf("%s/%s", topic, strings.ToLower(n[:1])+n[1:]), retained, val.Field(i).Interface())
			}
		}

	case reflect.Pointer:
		if reflect.ValueOf(payload).IsNil() {
			payload = nil
		}
		fallthrough

	default:
		m.publishSingleValue(topic, retained, payload)
	}
}

func (m *MQTT) publishString(topic string, retained bool, payload string) {
	token := m.Handler.Client.Publish(topic, m.Handler.Qos, retained, m.encode(payload))
	go m.Handler.WaitForToken("send", topic, token)
}

func (m *MQTT) publishSingleValue(topic string, retained bool, payload interface{}) {
	m.publisher(topic, retained, m.encode(payload))
}

func (m *MQTT) publish(topic string, retained bool, payload interface{}) {
	// publish phase values
	if slice, ok := payload.([]float64); ok && len(slice) == 3 {
		var total float64
		for i, v := range slice {
			total += v
			m.publishSingleValue(fmt.Sprintf("%s/l%d", topic, i+1), retained, v)
		}

		// publish sum value
		m.publishSingleValue(topic, retained, total)

		return
	}

	m.publishComplex(topic, retained, payload)
}

func (m *MQTT) Listen(site site.API) error {
	if err := m.listenSiteSetters(m.root+"/site", site); err != nil {
		return err
	}

	// loadpoint setters
	for id, lp := range site.Loadpoints() {
		topic := fmt.Sprintf("%s/loadpoints/%d", m.root, id+1)
		if err := m.listenLoadpointSetters(topic, site, lp); err != nil {
			return err
		}
	}

	// vehicle setters
	for _, vehicle := range site.Vehicles().Settings() {
		topic := fmt.Sprintf("%s/vehicles/%s", m.root, vehicle.Name())
		if err := m.listenVehicleSetters(topic, vehicle); err != nil {
			return err
		}
	}

	return nil
}

func (m *MQTT) listenSiteSetters(topic string, site site.API) error {
	for _, s := range []setter{
		{"/prioritySoc", floatSetterErr(site.SetPrioritySoc)},
		{"/bufferSoc", floatSetterErr(site.SetBufferSoc)},
		{"/bufferStartSoc", floatSetterErr(site.SetBufferStartSoc)},
		{"/residualPower", floatSetterErr(site.SetResidualPower)},
		{"/smartCostLimit", floatSetterErr(site.SetSmartCostLimit)},
	} {
		if err := m.Handler.ListenSetter(topic+s.topic, s.fun); err != nil {
			return err
		}
	}

	return nil
}

func (m *MQTT) listenLoadpointSetters(topic string, site site.API, lp loadpoint.API) error {
	for _, s := range []setter{
		{"/mode", setterFunc(api.ChargeModeString, lp.SetMode)},
		{"/phases", intSetterErr(lp.SetPhases)},
		{"/limitSoc", intSetter(lp.SetLimitSoc)},
		{"/minCurrent", floatSetterErr(lp.SetMinCurrent)},
		{"/maxCurrent", floatSetterErr(lp.SetMaxCurrent)},
		{"/limitEnergy", floatSetter(lp.SetLimitEnergy)},
		{"/enableThreshold", floatSetter(lp.SetEnableThreshold)},
		{"/disableThreshold", floatSetter(lp.SetDisableThreshold)},
		{"/planEnergy", func(payload string) error {
			var plan struct {
				Time  time.Time `json:"time"`
				Value float64   `json:"value"`
			}
			err := json.Unmarshal([]byte(payload), &plan)
			if err == nil {
				err = lp.SetPlanEnergy(plan.Time, plan.Value)
			}
			return err
		}},
		{"/vehicle", func(payload string) error {
			// https://github.com/evcc-io/evcc/issues/11184 empty payload is swallowed by listener
			if payload == "-" {
				lp.SetVehicle(nil)
				return nil
			}
			vehicle, err := site.Vehicles().ByName(payload)
			if err == nil {
				lp.SetVehicle(vehicle.Instance())
			}
			return err
		}},
	} {
		if err := m.Handler.ListenSetter(topic+s.topic, s.fun); err != nil {
			return err
		}
	}

	return nil
}

func (m *MQTT) listenVehicleSetters(topic string, v vehicle.API) error {
	for _, s := range []setter{
		{topic + "/limitSoc", intSetter(v.SetLimitSoc)},
		{topic + "/minSoc", intSetter(v.SetMinSoc)},
		{topic + "/planSoc", func(payload string) error {
			var plan struct {
				Time  time.Time `json:"time"`
				Value int       `json:"value"`
			}
			err := json.Unmarshal([]byte(payload), &plan)
			if err == nil {
				err = v.SetPlanSoc(plan.Time, plan.Value)
			}
			return err
		}},
	} {
		if err := m.Handler.ListenSetter(s.topic, s.fun); err != nil {
			return err
		}
	}

	return nil
}

// Run starts the MQTT publisher for the MQTT API
func (m *MQTT) Run(site site.API, in <-chan util.Param) {
	// number of loadpoints
	topic := fmt.Sprintf("%s/loadpoints", m.root)
	m.publish(topic, true, len(site.Loadpoints()))

	// number of vehicles
	topic = fmt.Sprintf("%s/vehicles", m.root)
	m.publish(topic, true, len(site.Vehicles().Settings()))

	// TODO remove deprecated topics
	for _, dep := range deprecatedTopics {
		m.publish(fmt.Sprintf("%s/site/%s", m.root, dep), true, nil)
	}

	for id := range site.Loadpoints() {
		topic := fmt.Sprintf("%s/loadpoints/%d", m.root, id+1)
		for _, dep := range deprecatedTopics {
			m.publish(fmt.Sprintf("%s/%s", topic, dep), true, nil)
		}
	}

	for i := 0; i < 10; i++ {
		m.publish(fmt.Sprintf("%s/site/pv/%d", m.root, i), true, nil)
		m.publish(fmt.Sprintf("%s/site/battery/%d", m.root, i), true, nil)
		m.publish(fmt.Sprintf("%s/site/vehicles/%d", m.root, i), true, nil)
	}

	// alive indicator
	var updated time.Time

	// publish
	for p := range in {
		switch {
		case p.Loadpoint != nil:
			id := *p.Loadpoint + 1
			topic = fmt.Sprintf("%s/loadpoints/%d/%s", m.root, id, p.Key)
		case p.Key == "vehicles":
			topic = fmt.Sprintf("%s/vehicles", m.root)
		default:
			topic = fmt.Sprintf("%s/site/%s", m.root, p.Key)
		}

		// alive indicator
		if time.Since(updated) > time.Second {
			updated = time.Now()
			m.publish(fmt.Sprintf("%s/updated", m.root), true, updated.Unix())
		}

		// value
		m.publish(topic, true, p.Val)
	}
}

// parseFloat rejects NaN and Inf values
func parseFloat(payload string) (float64, error) {
	f, err := strconv.ParseFloat(payload, 64)
	if err == nil && (math.IsNaN(f) || math.IsInf(f, 0)) {
		err = fmt.Errorf("invalid float value: %s", payload)
	}
	return f, err
}
