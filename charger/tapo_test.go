package charger

import "testing"

func TestTapoHandshake(t *testing.T) {
	tp := &Tapo{
		uri:      "http://192.168.178.114/app",
		email:    "m.thierolf@googlemail.com",
		password: "tapo1234",
	}

	err := tp.TapoHandshake()

	t.Errorf("MobileConnect.login() error:\n%v", err)

}
