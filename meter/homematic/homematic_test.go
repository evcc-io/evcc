package homematic

import (
	"encoding/xml"
	"fmt"
	"strings"
	"testing"
)

func TestHMCall(t *testing.T) {

	c := NewConnection("192.168.178.98:2010", "Admin", "85cMvmeHFVJNk6z", "0001DD89AAD848", "6", "2")

	res, err := c.XmlCmd("getParamset", "0001DD89AAD848:6", "VALUES", "")
	if err != nil {
		t.Errorf("\nError +++\n%v\n", err)
	}
	t.Errorf("\nOK +++\n%v\n", res)
}

func TestGetSwitchState(t *testing.T) {

	c := NewConnection("192.168.178.98:2010", "Admin", "85cMvmeHFVJNk6z", "0001DD89AAD848", "6", "3")

	res, err := c.Enabled()
	if err != nil {
		t.Errorf("\nError +++\n%v\n", err)
	}
	t.Errorf("\nOK +++\n%v\n", res)
}

func TestGetMeterPower(t *testing.T) {

	c := NewConnection("192.168.178.98:2010", "Admin", "85cMvmeHFVJNk6z", "0001DD89AAD848", "6", "3")

	res, err := c.CurrentPower()
	if err != nil {
		t.Errorf("\nError +++\n%v\n", err)
	}
	t.Errorf("\nOK +++\n%v\n", res)
}

func TestXMLUnmarshall(t *testing.T) {

	contents := `<?xml version="1.0" encoding="ISO-8859-1"?><methodResponse><params><param><value><boolean>1</boolean></value></param></params></methodResponse>`

	m := &MethodResponse{}

	xml.Unmarshal([]byte(strings.Replace(contents, "ISO-8859-1", "UTF-8", 1)), &m)

	fmt.Printf("%v\n", m.Value.BoolValue)

	t.Errorf("\nOK +++\n%#v\n", m)
	//t.Errorf("\n+++\n%#v\n", members)

}
