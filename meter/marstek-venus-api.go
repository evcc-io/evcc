package meter

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/marstekvenusapi"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/spf13/cast"
)

const (
	udpTimeout = time.Second * 20
)

// MarstekVenusApi meter implementation
type MarstekVenusApi struct {
	log  *util.Logger
	conn string
	//usage   templates.Usage
	timeout time.Duration
	recv    chan marstekvenusapi.UDPMsg
	sender  *marstekvenusapi.Sender
	tracker *marstekvenusapi.RequestTracker
}

func init() {
	registry.Add("marstek-venus-api", NewMarstekVenusApiFromConfig)
}

//go:generate go tool decorate -f decorateMarstekVenusApi  -b *MarstekVenusApi -r api.Meter   -t "api.Battery,Soc,func() (float64, error)"   -t "api.BatteryCapacity,Capacity,func() float64"  -t "api.BatteryController,SetBatteryMode,func(api.BatteryMode) error"

//	Battery, Soc						Bat.GetStatus -> r.soc
//	BatteryCapacity, Capacity			Bat.GetStatus -> r.rated_cap
//	BatteryController, SetBatteryMode	ES.SetMode(config.mode:="Auto" | "Passive"
//	Meter, CurrentPower					ES.GetStatus -> r.result.	Bei diesem Request gibt es einen Timeout
//		        							• ongrid_power
//		       								• Offgrid_power
//		       								• Bat_power

// NewMarstekVenusApiFromConfig creates a Marstek Battery meter from generic config
func NewMarstekVenusApiFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		batteryCapacity `mapstructure:",squash"`
		Usage           templates.Usage
		Uri             string
		Timeout         time.Duration
	}{
		Timeout: udpTimeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewMarstekVenusApi(cc.Uri, cc.Timeout, cc.batteryCapacity.Decorator())
}

// NewMarstekVenusApi creates Marstek Open Api based Meter and Battery Control
func NewMarstekVenusApi(uri string, timeout time.Duration, capacity func() float64) (api.Meter, error) {
	log := util.NewLogger("marstek-venus-api")
	tracker := marstekvenusapi.NewRequestTracker()

	listenerinstance, err := marstekvenusapi.Instance(log, tracker)
	if err != nil {
		return nil, err
	}

	conn := util.DefaultPort(uri, marstekvenusapi.DEFAULT_PORT)
	sender, err := marstekvenusapi.NewSender(log, conn, tracker)

	if err != nil {
		return nil, err
	}

	m := &MarstekVenusApi{
		log:     log,
		conn:    conn,
		timeout: timeout,
		recv:    make(chan marstekvenusapi.UDPMsg),
		sender:  sender,
		tracker: tracker,
	}

	// add recv channel to the list of listeners to receive incoming IDPMsg
	listenerinstance.Subscribe(conn, m.recv)

	// decorate battery
	var (
		batteryCapacity func() float64
		batterySoc      func() (float64, error)
		batteryMode     func(api.BatteryMode) error
	)

	batteryCapacity = capacity
	batterySoc = m.batterySoc
	batteryMode = m.setBatteryMode

	return decorateMarstekVenusApi(m, batterySoc, batteryCapacity, batteryMode), nil
}

func (m *MarstekVenusApi) roundtrip(rtype marstekvenusapi.RequestType, req interface{}, res interface{}) error {
	// 1) send
	time.Sleep(time.Second * 3)
	id, err := m.sender.SendMtekReq(rtype, req)
	if err != nil {
		return err
	}
	// 2) wait on m.recv or timeout
	timer := time.NewTimer(m.timeout)
	defer timer.Stop()
	for {
		select {
		case udpMsg := <-m.recv:
			// unpack wrapper
			var wrapper marstekvenusapi.Response
			if err := json.Unmarshal(udpMsg.Message, &wrapper); err != nil {
				return err
			}
			// skip not-my-ID
			if wrapper.ID != id {
				continue
			}
			// RPC error?
			if wrapper.Error != nil {
				return fmt.Errorf("rpc error: %s", wrapper.Error.Message)
			}
			// return the unwrapped result
			return json.Unmarshal(wrapper.Result, res)
		case <-timer.C:
			return fmt.Errorf("rpc timeout after %s", m.timeout)
		}
	}
}

// CurrentPower implements the api.Meter interface
func (m *MarstekVenusApi) CurrentPower() (float64, error) {
	// send ES.GetStatus ... likely not to work due to FW issues causing timeout
	var mvar marstekvenusapi.EsStatusResult

	err := m.roundtrip(marstekvenusapi.METHOD_ES_STATUS, nil, &mvar)
	if err != nil {
		return 0, err
	}
	retVal := mvar.OnGridPower - mvar.OffGridPower
	m.log.TRACE.Printf("total power: %d W", retVal)
	return cast.ToFloat64E(retVal)
}

var _ api.MeterEnergy = (*MarstekVenusApi)(nil)

func (m *MarstekVenusApi) batterySoc() (float64, error) {
	//var mvar marstekvenusapi.BatStatusResult
	var mvar marstekvenusapi.EsModeResult

	// err := m.roundtrip(marstekvenusapi.METHOD_BATTERY_STATUS, nil, &mvar)
	err := m.roundtrip(marstekvenusapi.METHOD_ES_MODE, nil, &mvar)
	if err != nil {
		return 0, err
	}
	retVal := mvar.BatSoc
	m.log.TRACE.Printf("got soc of %d percent", retVal)
	return cast.ToFloat64E(retVal)
}

// TotalEnergy implements the api.MeterEnergy interface
// MeterEnergy provides total energy in kWh
func (m *MarstekVenusApi) TotalEnergy() (float64, error) {
	var mvar marstekvenusapi.EsStatusResult

	err := m.roundtrip(marstekvenusapi.METHOD_ES_STATUS, nil, &mvar)
	if err != nil {
		return 0, err
	}
	retVal := mvar.TotalGridInputEnergy + mvar.TotalGridOutputEnergy
	m.log.TRACE.Printf("total energy: %d kWh", retVal)
	return cast.ToFloat64E(retVal)
}

func (m *MarstekVenusApi) setBatteryMode(mode api.BatteryMode) error {
	return api.ErrNotAvailable
}

var _ api.Diagnosis = (*MarstekVenusApi)(nil)

// Diagnose implements the api.Diagnosis interface
func (m *MarstekVenusApi) Diagnose() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	var mvar marstekvenusapi.DeviceResult

	err := m.roundtrip(marstekvenusapi.METHOD_GET_DEVICE, nil, &mvar)

	if err != nil {
		return
	}

	fmt.Fprintf(w, "  IP:\t%s\n", mvar.IpAddr)
	fmt.Fprintf(w, "  Serial:\t%s\n", mvar.Device)
	fmt.Fprintf(w, "  S/W Version:\t%d\n", mvar.Version)
	fmt.Fprintf(w, "  BLE Mac:\t%s\n", mvar.BleMac)
	fmt.Fprintf(w, "  Wifi Mac:\t%s\n", mvar.WifiMac)
	fmt.Fprintf(w, "  SSID:\t%s\n", mvar.WifiName)

	fmt.Fprintln(w)

	w.Flush()
}
