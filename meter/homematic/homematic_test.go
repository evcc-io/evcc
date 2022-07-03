package homematic

import (
	"testing"
)

func TestGetSwitchState(t *testing.T) {

	c, err := NewConnection("192.168.178.98:2010", "0001DD89AAD848", "6", "3", "Admin", "85cMvmeHFVJNk6z")

	res, err := c.Enabled()
	if err != nil {
		t.Errorf("\nError +++\n%v\n", err)
	}
	t.Errorf("\nOK +++\n%v\n", res)
}

func TestGetMeterPower(t *testing.T) {

	c, err := NewConnection("192.168.178.98:2010", "0001DD89AAD848", "6", "3", "Admin", "85cMvmeHFVJNk6z")

	res, err := c.CurrentPower()
	if err != nil {
		t.Errorf("\nError +++\n%v\n", err)
	}
	t.Errorf("\nOK +++\n%v\n", res)
}

func TestEnable(t *testing.T) {

	c, err := NewConnection("192.168.178.98:2010", "0001DD89AAD848", "6", "3", "Admin", "85cMvmeHFVJNk6z")

	err = c.Enable(false)
	if err != nil {
		t.Errorf("\nError +++\n%v\n", err)
	}
	t.Errorf("\nOK +++\n%v\n", "")
}
