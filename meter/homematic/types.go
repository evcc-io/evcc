package homematic

import (
	"encoding/xml"
)

type MethodGetParam struct {
	CCUString string `xml:"value>string,omitempty"`
}

type MethodGetCall struct {
	XMLName    xml.Name         `xml:"methodCall"`
	MethodName string           `xml:"methodName"`
	Params     []MethodGetParam `xml:"params>param,omitempty"`
}

type ParamValue struct {
	CCUString string `xml:"value>string,omitempty,"`
	CCUBool   string `xml:"value>boolean,omitempty,"`
}

type MethodSetCall struct {
	XMLName    xml.Name     `xml:"methodCall"`
	MethodName string       `xml:"methodName"`
	Params     []ParamValue `xml:"params>param,omitempty"`
}

type MethodResponseValue struct {
	XMLName   xml.Name `xml:"value"`
	CCUBool   int64    `xml:"boolean"`
	CCUFloat  float64  `xml:"double"`
	CCUInt    int64    `xml:"i4"`
	CCUString string   `xml:"string"`
}

type MethodResponse struct {
	XMLName xml.Name            `xml:"methodResponse"`
	Value   MethodResponseValue `xml:"params>param>value,omitempty"`
}
