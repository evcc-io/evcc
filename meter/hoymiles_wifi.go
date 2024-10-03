package meter

import (
	"net"
	"os"
	"syscall"
	"time"

	"github.com/BLun78/hoymiles_wifi"
	"github.com/BLun78/hoymiles_wifi/hoymiles/models"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.Add("hoymiles-wifi", NewHoymilesWifiMeterFromConfig)
}

type HoymilesWifi struct {
	client           *hoymiles_wifi.ClientData
	log              *util.Logger
	host string
	lastValue        float64
	lastValueUpdated time.Time
}

func NewHoymilesWifiMeterFromConfig(other map[string]interface{}) (api.Meter, error) {
	var cc struct {
		Host string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("hoymiles-wifi")
	log.TRACE.Printf("Start HoymilesWifi setup: %s", cc.Host)

	client := hoymiles_wifi.NewClientDefault(cc.Host)
	client.ConnectionTimeout = 5 * time.Second

	return &HoymilesWifi{
		client:    client,
		log:       log,
		cc:        cc,
	}, nil
}

// CurrentPower implements the api.Meter interface
func (hmWifi *HoymilesWifi) CurrentPower() (float64, error) {
	var value float64
	request := &models.RealDataNewReqDTO{}
	// int32 would not be Year 2038 safe
	// See https://en.wikipedia.org/wiki/Year_2038_problem
	// Not 100% sure if the models are self-defined or provided by hoymiles
	request.Time = int32(time.Now().Unix())
	request.TimeYmdHms = time.Now().Format("2006-01-02 15:04:05")

	result, err := hmWifi.client.GetRealDataNew(request)
	if err != nil {
		if hmWifi.lastValue != 0 && !hmWifi.lastValueUpdated.Add(time.Minute*15).Before(time.Now()) {
			hmWifi.lastValue = 0
		}
		if err.Error() == "client connection is closed" {
			hmWifi.log.DEBUG.Printf("HoymilesWifi the Host is offline: %s", hmWifi.cc.Host)
			return hmWifi.lastValue, nil
		}
		opErr, ok := err.(*net.OpError)
		if ok {
			sysErr, ok2 := opErr.Err.(*os.SyscallError)
			if ok2 && sysErr.Err == syscall.Errno(10060) {
				hmWifi.log.DEBUG.Printf("HoymilesWifi the Host is offline: %s", hmWifi.cc.Host)
				return hmWifi.lastValue, nil
			}
		}
		return value, err
	}

	defer func(client *hoymiles_wifi.ClientData) {
		_ = client.CloseConnection()
	}(hmWifi.client)

	if result.DtuPower > 0 {
		value = float64(result.DtuPower) / 10
	}

	hmWifi.lastValue = value
	hmWifi.lastValueUpdated = time.Now()

	return float64(result.DtuPower) / 10, nil
}
