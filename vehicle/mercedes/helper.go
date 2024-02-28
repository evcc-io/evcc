package mercedes

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"
)

// Helper provides utility primitives
type Helper struct {
	*http.Client
}

var (
	mu         sync.Mutex
	identities = make(map[string]*Identity)
)

func getInstance(subject string) *Identity {
	v := identities[subject]
	return v
}

func addInstance(subject string, identity *Identity) {
	identities[subject] = identity
}

const (
	BffUriEMEA                 = "https://bff.emea-prod.mobilesdk.mercedes-benz.com"
	WidgetUriEMEA              = "https://widget.emea-prod.mobilesdk.mercedes-benz.com"
	BffUriAPAC                 = "https://bff.amap-prod.mobilesdk.mercedes-benz.com"
	WidgetUriAPAC              = "https://widget.amap-prod.mobilesdk.mercedes-benz.com"
	BffUriNORAM                = "https://bff.amap-prod.mobilesdk.mercedes-benz.com"
	WidgetUriNORAM             = "https://widget.amap-prod.mobilesdk.mercedes-benz.com"
	IdUri                      = "https://id.mercedes-benz.com"
	ClientId                   = "01398c1c-dc45-4b42-882b-9f5ba9f175f1"
	RisApplicationVersionEMEA  = "1.40.0"
	RisSdkVersionEMEA          = "2.111.1"
	RisApplicationVersionAPAC  = "1.40.0"
	RisSdkVersionAPAC          = "2.111.1"
	RisApplicationVersionNORAM = "3.40.0"
	RisSdkVersionNORAM         = "2.111.1"
	RisOsVersion               = "17.3"
	RisOsName                  = "ios"
	XApplicationNameEMEA       = "mycar-store-ece"
	XApplicationNameAPAC       = "mycar-store-ap"
	XApplicationNameNORAM      = "mycar-store-us"
	UserAgent                  = "MyCar/%s (com.daimler.ris.mercedesme.%s.ios; %s %s) Alamofire/5.4.0"
	UserAgentAPAC              = "mycar-store-ap v%s, %s %s, SDK %s"
	Locale                     = "en-GB"
	CountryCode                = "EN"
)

func getBffUri(region string) string {
	switch region {
	case "EMEA":
		return BffUriEMEA
	case "APAC":
		return BffUriAPAC
	case "NORAM":
		return BffUriNORAM
	}
	return BffUriEMEA
}

func getWidgetUri(region string) string {
	switch region {
	case "EMEA":
		return WidgetUriEMEA
	case "APAC":
		return WidgetUriAPAC
	case "NORAM":
		return WidgetUriNORAM
	}
	return WidgetUriEMEA
}

func mbheaders(includeAuthServerHeader bool, region string) map[string]string {
	headers := map[string]string{
		"Ris-Os-Name":     RisOsName,
		"Ris-Os-Version":  RisOsVersion,
		"X-Locale":        Locale,
		"X-Trackingid":    uuid.New().String(),
		"X-Sessionid":     uuid.New().String(),
		"Content-Type":    "application/json",
		"Accept-Language": "en-GB",
		"Accept":          "*/*",
		"x-dev":           "1",
	}

	switch region {
	case "EMEA":
		headers["Ris-Sdk-Version"] = RisSdkVersionEMEA
		headers["Ris-Application-Version"] = RisApplicationVersionEMEA
		headers["X-Applicationname"] = XApplicationNameEMEA
		headers["User-Agent"] = fmt.Sprintf(UserAgent, RisApplicationVersionEMEA, "ece", RisOsName, RisOsVersion)
	case "APAC":
		headers["Ris-Sdk-Version"] = RisSdkVersionAPAC
		headers["Ris-Application-Version"] = RisApplicationVersionAPAC
		headers["X-Applicationname"] = XApplicationNameAPAC
		headers["User-Agent"] = fmt.Sprintf(UserAgentAPAC, RisApplicationVersionAPAC, RisOsName, RisOsVersion, RisSdkVersionAPAC)
	case "NORAM":
		headers["Ris-Sdk-Version"] = RisSdkVersionNORAM
		headers["Ris-Application-Version"] = RisApplicationVersionNORAM
		headers["X-Applicationname"] = XApplicationNameNORAM
		headers["User-Agent"] = fmt.Sprintf(UserAgent, RisApplicationVersionEMEA, "ece", RisOsName, RisOsVersion)
	}

	if includeAuthServerHeader {
		headers["Stage"] = "prod"
		headers["X-Device-Id"] = uuid.New().String()
		headers["X-Request-Id"] = uuid.New().String()
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	}

	return headers
}
