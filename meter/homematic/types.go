package homematic

import (
	"encoding/xml"
)

// Homematic CCU XML-RPC types
// https://homematic-ip.com/sites/default/files/downloads/HM_XmlRpc_API.pdf
// https://homematic-ip.com/sites/default/files/downloads/HMIP_XmlRpc_API_Addendum.pdf

type Param struct {
	CCUBool   string  `xml:"value>boolean,omitempty"`
	CCUFloat  float64 `xml:"value>double,omitempty"`
	CCUInt    int64   `xml:"value>i4,omitempty"`
	CCUString string  `xml:"value>string,omitempty"`
}

type MethodCall struct {
	XMLName    xml.Name `xml:"methodCall"`
	MethodName string   `xml:"methodName"`
	Params     []Param  `xml:"params>param,omitempty"`
}

type Member struct {
	Name  string     `xml:"name,omitempty"`
	Value FaultValue `xml:"value,omitempty"`
}

type FaultValue struct {
	XMLName   xml.Name `xml:"value"`
	CCUString string   `xml:",chardata"`
	CCUInt    int64    `xml:"i4,omitempty"`
}

type MethodResponse struct {
	XMLName xml.Name `xml:"methodResponse"`
	Value   Param    `xml:"params>param,omitempty"`
	Fault   []Member `xml:"fault>value>struct>member,omitempty"`
}
