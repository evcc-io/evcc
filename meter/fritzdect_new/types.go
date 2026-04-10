package fritzdect_new

import (
	"encoding/xml"
	"time"

	"github.com/evcc-io/evcc/util/request"
)

// FRITZ! FritzBox AHA interface and authentication specifications:
// https://fritz.com/fileadmin/user_upload/Global/Service/Schnittstellen/AHA-HTTP-Interface.pdf
// https://fritz.com/fileadmin/user_upload/Global/Service/Schnittstellen/AVM_Technical_Note_-_Session_ID.pdf

// FritzDECT settings
type Settings struct {
	URI, AIN, User, Password string
}

// FritzDECT connection
type Connection struct {
	*request.Helper
	*Settings
	SID     string
	updated time.Time
}

// https://fritz.com/fileadmin/user_upload/Global/Service/Schnittstellen/AVM_Technical_Note_-_Session_ID_english_2021-05-03.pdf
const sessionTimeout = 15 * time.Minute

// Devicestats structures getbasicdevicesstats command response (AHA-HTTP-Interface)
type Devicestats struct {
	XMLName xml.Name `xml:"devicestats"`
	Energy  Energy   `xml:"energy"`
	Power   Power    `xml:"power"`
	Voltage Voltage  `xml:"voltage"`
}

// Energy structures getbasicdevicesstats command energy response (AHA-HTTP-Interface)
type Energy struct {
	XMLName xml.Name `xml:"energy"`
	Values  []string `xml:"stats"`
}

// Energy structures getbasicdevicesstats command energy response (AHA-HTTP-Interface)
type Voltage struct {
	XMLName xml.Name `xml:"voltage"`
	Values  []string `xml:"stats"`
}

// Energy structures getbasicdevicesstats command energy response (AHA-HTTP-Interface)
type Power struct {
	XMLName xml.Name `xml:"power"`
	Values  []string `xml:"stats"`
}
