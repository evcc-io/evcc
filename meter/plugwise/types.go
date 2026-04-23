package plugwise

import "encoding/xml"

// DomainObjects is the root element of the Plugwise Smile P1 /core/domain_objects response.
type DomainObjects struct {
	XMLName  xml.Name `xml:"domain_objects"`
	Location Location `xml:"location"`
}

// Location represents the building/home location element.
type Location struct {
	Logs Logs `xml:"logs"`
}

// Logs contains the measurement log entries for the location.
type Logs struct {
	PointLogs []PointLog `xml:"point_log"`
}

// PointLog represents a single point (instantaneous) measurement log entry.
// cumulative_log and interval_log elements are ignored by this struct (different element names).
type PointLog struct {
	Type   string `xml:"type"`
	Unit   string `xml:"unit"`
	Period Period `xml:"period"`
}

// Period holds the measurement values within a log entry.
type Period struct {
	Measurements []Measurement `xml:"measurement"`
}

// Measurement is a single measured value, optionally tagged with a tariff (nl_peak/nl_offpeak).
type Measurement struct {
	Tariff string  `xml:"tariff,attr"`
	Value  float64 `xml:",chardata"`
}

// PowerWatts sums all measurement values for the given log type where unit is "W".
// Both tariff measurements (nl_peak + nl_offpeak) are summed.
// Returns 0.0 if the log type is not present or has no measurements with unit "W".
func (logs Logs) PowerWatts(typeName string) float64 {
	var total float64
	for _, pl := range logs.PointLogs {
		if pl.Type == typeName && pl.Unit == "W" {
			for _, m := range pl.Period.Measurements {
				total += m.Value
			}
		}
	}
	return total
}

// VoltageVolts returns the measurement value for the given log type where unit is "V".
// Voltage point_log entries have a single measurement (no tariff attribute); summing
// is correct and symmetric with PowerWatts.
// Returns 0.0 if the log type is not present or has no measurements with unit "V".
func (logs Logs) VoltageVolts(typeName string) float64 {
	var total float64
	for _, pl := range logs.PointLogs {
		if pl.Type == typeName && pl.Unit == "V" {
			for _, m := range pl.Period.Measurements {
				total += m.Value
			}
		}
	}
	return total
}
