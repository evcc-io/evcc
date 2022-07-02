package homematic

import (
	"encoding/xml"
)

type MethodParam struct {
	ParamString string `xml:"value>string,omitempty"`
}

type MethodCall struct {
	XMLName    xml.Name      `xml:"methodCall"`
	MethodName string        `xml:"methodName"`
	Params     []MethodParam `xml:"params>param,omitempty"`
}

type MethodResponseValue struct {
	XMLName     xml.Name `xml:"value"`
	BoolValue   int64    `xml:"boolean"`
	FloatValue  float64  `xml:"double"`
	IntValue    int64    `xml:"i4"`
	StringValue string   `xml:"string"`
}

type MethodResponse struct {
	XMLName xml.Name            `xml:"methodResponse"`
	Value   MethodResponseValue `xml:"params>param>value,omitempty"`
}
