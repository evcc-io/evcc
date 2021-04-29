package fritzdect

import "encoding/xml"

// Devicestats structures getbasicdevicesstats command response (AHA-HTTP-Interface)
type Devicestats struct {
	XMLName xml.Name `xml:"devicestats"`
	Energy  Energy   `xml:"energy"`
}

// Energy structures getbasicdevicesstats command energy response (AHA-HTTP-Interface)
type Energy struct {
	XMLName xml.Name `xml:"energy"`
	Values  []string `xml:"stats"`
}
