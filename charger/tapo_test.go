package charger

import (
	"testing"

	"github.com/evcc-io/evcc/charger/tapo"
)

func TestLogin(t *testing.T) {
	device := tapo.NewConnection("192.168.178.114", "m.thierolf@googlemail.com", "tapo1234")

	if err := device.Login(); err != nil {
		t.Errorf("Login:\n%v\n", err)
	}

	// deviceResponse, err := device.ExecMethod("get_device_info", false)
	// deviceResponse, err := device.ExecMethod("set_device_info", false)
	deviceResponse, err := device.ExecMethod("get_energy_usage", false)
	if err != nil {
		t.Errorf("deviceResponse:\n%v\nerror:\n%v", deviceResponse, err)
	}

	t.Errorf("\ndeviceResponse:\n%v\n", deviceResponse)
	//t.Errorf("\ndeviceON:\n%v\n", deviceResponse.Result.DeviceON)
	t.Errorf("\nMAC:\n%v\ndeviceON:\n%v\nCurrent_Power:\n%d\n", deviceResponse.Result.MAC, deviceResponse.Result.DeviceON, deviceResponse.Result.Current_Power)
	// device.Switch(true)

}
