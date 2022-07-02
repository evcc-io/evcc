package homematic

import (
	"encoding/xml"
)

type ParamValue struct {
	CCUBool   string  `xml:"value>boolean,omitempty"`
	CCUFloat  float64 `xml:"value>double,omitempty"`
	CCUInt    int64   `xml:"value>i4,omitempty"`
	CCUString string  `xml:"value>string,omitempty"`
}

type MethodCall struct {
	XMLName    xml.Name     `xml:"methodCall"`
	MethodName string       `xml:"methodName"`
	Params     []ParamValue `xml:"params>param,omitempty"`
}

type MethodResponse struct {
	XMLName xml.Name   `xml:"methodResponse"`
	Value   ParamValue `xml:"params>param,omitempty"`
}
