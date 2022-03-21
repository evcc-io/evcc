package charger

import (
	"testing"

	"github.com/evcc-io/evcc/charger/tapo"
)

func TestTapoHandshake(t *testing.T) {
	tp := &Tapo{
		uri:      "192.168.178.114",
		email:    "m.thierolf@googlemail.com",
		password: "tapo1234",
	}

	err := tp.Handshake()
	if err != nil {
		t.Errorf("TapoHandshake:\n%v\n", err)
	}
}

func TestTapoLogin(t *testing.T) {
	tp := &Tapo{
		uri:      "192.168.178.114",
		email:    "m.thierolf@googlemail.com",
		password: "tapo1234",
	}

	err := tp.Handshake()
	if err != nil {
		t.Errorf("TapoHandshake:\n%v\n", err)
	}

	err = tp.TapoLogin()
	t.Errorf("TapoLogin:\n%v\n", err)
}

func TestLogin(t *testing.T) {
	device := tapo.New("192.168.178.114", "m.thierolf@googlemail.com", "tapo1234")

	if err := device.Handshake(); err != nil {
		t.Errorf("Handshake:\n%v\n", err)
	}

	if err := device.Login(); err != nil {
		t.Errorf("Login:\n%v\n", err)
	}

	deviceInfo, err := device.GetDeviceInfo()
	if err != nil {
		t.Errorf("deviceInfo:\n%v\nerror:\n%v", deviceInfo, err)
	}

	t.Errorf("\ndeviceON:\n%v\n", deviceInfo.Result.DeviceON)
	// device.Switch(true)

}
