package charger

import (
	"testing"

	"github.com/evcc-io/evcc/charger/tapo"
)

func TestLogin(t *testing.T) {
	device := tapo.NewConnection("192.168.178.114", "m.thierolf@googlemail.com", "tapo1234")

	if err := device.Handshake(); err != nil {
		t.Errorf("Handshake:\n%v\n", err)
	}

	if err := device.Login(); err != nil {
		t.Errorf("Login:\n%v\n", err)
	}

	// deviceResponse, err := device.ExecMethod("set_device_info", true)
	deviceResponse, err := device.ExecMethod("get_device_info", false)
	if err != nil {
		t.Errorf("deviceResponse:\n%v\nerror:\n%v", deviceResponse, err)
	}

	t.Errorf("\ndeviceResponse:\n%v\n", deviceResponse)
	//t.Errorf("\ndeviceON:\n%v\n", deviceResponse.Result.DeviceON)
	t.Errorf("\nMAC:\n%v\ndeviceON:\n%v\n", deviceResponse.Result.MAC, deviceResponse.Result.DeviceON)
	// device.Switch(true)

}
