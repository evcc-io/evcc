package fritzdect

import "encoding/xml"

// getdevicestats command response (AHA-HTTP-Interface)
type Devicestats struct {
	XMLName xml.Name `xml:"devicestats"`
	Energy  Energy   `xml:"energy"`
}

type Energy struct {
	XMLName xml.Name `xml:"energy"`
	Values  []string `xml:"stats"`
}
