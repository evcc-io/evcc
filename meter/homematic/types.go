package homematic

import (
	"encoding/xml"
	"fmt"
)

// Homematic CCU XML-RPC types
// https://homematic-ip.com/sites/default/files/downloads/HM_XmlRpc_API.pdf
// https://homematic-ip.com/sites/default/files/downloads/HMIP_XmlRpc_API_Addendum.pdf

type MethodCall struct {
	XMLName    xml.Name `xml:"methodCall"`
	MethodName string   `xml:"methodName"`
	Params     []Param  `xml:"params>param,omitempty"`
}

type Param struct {
	CCUBool   string  `xml:"value>boolean,omitempty"`
	CCUFloat  float64 `xml:"value>double,omitempty"`
	CCUInt    int64   `xml:"value>i4,omitempty"`
	CCUString string  `xml:"value>string,omitempty"`
}
type Member struct {
	Name  string `xml:"name,omitempty"`
	Value Value  `xml:"value,omitempty"`
}

type MethodResponse struct {
	XMLName   xml.Name `xml:"methodResponse"`
	CCUBool   string   `xml:"params>param>value>boolean,omitempty"`
	CCUFloat  float64  `xml:"params>param>value>double,omitempty"`
	CCUInt    int64    `xml:"params>param>value>i4,omitempty"`
	CCUString string   `xml:"params>param>value>string,omitempty"`
	Member    []Member `xml:"params>param>value>struct>member,omitempty"`
	Fault     []Member `xml:"fault>value>struct>member,omitempty"`
}

// FloatValue selects a float value of a CCU API response member
func (res *MethodResponse) FloatValue(val string) float64 {
	for _, m := range res.Member {
		if m.Name == val {
			return m.Value.CCUFloat
		}
	}

	return 0
}

// BoolValue selects a float value of a CCU API response member
func (res *MethodResponse) BoolValue(val string) bool {
	for _, m := range res.Member {
		if m.Name == val {
			return m.Value.CCUBool
		}
	}

	return false
}

// Error checks on Homematic CCU error codes
// Refer to page 30 of https://homematic-ip.com/sites/default/files/downloads/HM_XmlRpc_API.pdf
func (res *MethodResponse) Error() error {
	var faultCode int64
	var faultString string

	for _, f := range res.Fault {
		if f.Name == "faultCode" {
			faultCode = f.Value.CCUInt
		}
		if f.Name == "faultString" {
			faultString = f.Value.CCUString
		}
	}

	if faultCode != 0 {
		if faultString == "" {
			faultString = "unknown api error"
		}

		return fmt.Errorf("%s (%v)", faultString, faultCode)
	}

	return nil
}

type Value struct {
	XMLName   xml.Name `xml:"value"`
	CCUString string   `xml:",chardata"`
	CCUInt    int64    `xml:"i4,omitempty"`
	CCUBool   bool     `xml:"boolean,omitempty"`
	CCUFloat  float64  `xml:"double,omitempty"`
}
