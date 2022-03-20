package charger

import "testing"

func TestTapoHandshake(t *testing.T) {
	tp := &Tapo{
		email:    "m.thierolf@googlemail.com",
		password: "tapo123",
	}

	err := tp.TapoHandshake()

	t.Errorf("MobileConnect.login() error:\n%v", err)

}
