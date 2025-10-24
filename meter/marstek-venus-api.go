package meter

import (
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/marstekvenusapi"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/templates"
)

const (
	udpTimeout = time.Second * 3
)

// MarstekVenusApi meter implementation
type MarstekVenusApi struct {
	log     *util.Logger
	conn    string
	usage   templates.Usage
	timeout time.Duration
	recv    chan marstekvenusapi.UDPMsg
	sender  *marstekvenusapi.Sender
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

	//cfg := rscp.ClientConfig{
	//		Address:           host,
	//		Port:              uint16(port),
	//		Username:          cc.User,
	//		Password:          cc.Password,
	//		Key:               cc.Key,
	//		ConnectionTimeout: cc.Timeout,
	//		SendTimeout:       cc.Timeout,
	//		ReceiveTimeout:    cc.Timeout,
	//	}

	return NewMarstekVenusApi(cc.Uri, cc.Timeout, cc.batteryCapacity.Decorator())
}

var mtekOnce sync.Once

// NewMarstekVenusApi creates Blueprint charger
func NewMarstekVenusApi(uri string, timeout time.Duration, capacity func() float64) (api.Meter, error) {
	log := util.NewLogger("marstek-venus-api")

	instance, err := marstekvenusapi.Instance(log)
	if err != nil {
		return nil, err
	}

	conn := util.DefaultPort(uri, marstekvenusapi.DEFAULT_PORT)
	sender, err := marstekvenusapi.NewSender(log, conn)

	m := &MarstekVenusApi{
		log:     log,
		conn:    conn,
		timeout: timeout,
		recv:    make(chan marstekvenusapi.UDPMsg),
		sender:  sender,
	}

	instance.Subscribe(conn, m.recv)

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

// CurrentPower implements the api.Meter interface
func (m *MarstekVenusApi) CurrentPower() (float64, error) {
	// send ES.GetStatus ... likely not to work due to FW issues causing timeout
	return 0, api.ErrNotAvailable
}

var _ api.MeterEnergy = (*MarstekVenusApi)(nil)

func (m *MarstekVenusApi) batterySoc() (float64, error) {

	return 0, api.ErrNotAvailable
}

// TotalEnergy implements the api.MeterEnergy interface
func (m *MarstekVenusApi) TotalEnergy() (float64, error) {
	return 0, api.ErrNotAvailable
}

func (m *MarstekVenusApi) setBatteryMode(mode api.BatteryMode) error {
	return api.ErrNotAvailable
}
