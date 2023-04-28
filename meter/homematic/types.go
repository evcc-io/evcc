package homematic

import (
	"encoding/xml"
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
type Value struct {
	XMLName   xml.Name `xml:"value"`
	CCUString string   `xml:",chardata"`
	CCUInt    int64    `xml:"i4,omitempty"`
	CCUBool   bool     `xml:"boolean,omitempty"`
	CCUFloat  float64  `xml:"double,omitempty"`
}
