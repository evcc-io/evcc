package mercedes

import (
	"fmt"
	"net/http"
	"sync"
	"uuid"
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
	RisApplicationVersionEMEA  = "1.65.1 (3174)"
	RisSdkVersionEMEA          = "4.4.2"
	RisApplicationVersionAPAC  = "1.65.0"
	RisSdkVersionAPAC          = "4.4.2"
	RisApplicationVersionNORAM = "3.65.1"
	RisSdkVersionNORAM         = "4.4.2"
	RisOsVersion               = "26.3"
	RisOsName                  = "ios"
	XApplicationNameEMEA       = "mycar-store-ece"
	XApplicationNameAPAC       = "mycar-store-ap"
	XApplicationNameNORAM      = "mycar-store-us"
	UserAgent                  = "MyCar/%s (com.daimler.ris.mercedesme.%s.ios; %s %s) Alamofire/5.4.0"
	UserAgentAPAC              = "mycar-store-ap v%s, %s %s, SDK %s"
	Locale                     = "en-GB"
	CountryCode                = "EN"

	// Websocket (VSU push) endpoints, one per account/region.
	WebsocketUriEMEA  = "wss://websocket.emea-prod.mobilesdk.mercedes-benz.com/v2/ws"
	WebsocketUriAPAC  = "wss://websocket.amap-prod.mobilesdk.mercedes-benz.com/v2/ws"
	WebsocketUriNORAM = "wss://websocket.amap-prod.mobilesdk.mercedes-benz.com/v2/ws"

	// RIS constants for the websocket handshake. These are coupled to the VSU
	// switch: only with sufficiently recent RIS headers does the backend push
	// typed vehicle_status_updates instead of the legacy vepUpdates string map.
	// Kept static (mirroring mbapi2020), bump together with the app.
	RisWsApplicationVersionEMEA  = "1.68.0 (3060)"
	RisWsApplicationVersionNORAM = "3.67.0"
	RisWsApplicationVersionAPAC  = "1.67.0"
	RisWsSdkVersion              = "4.10.0"
	WebsocketUserAgent           = "Mercedes-Benz/3044 CFNetwork/3860.400.22 Darwin/25.3.0"
	WebsocketUserAgentAPAC       = "mycar-store-ap 1.67.0, ios 26.3, SDK 4.10.0"
	WebsocketUserAgentNORAM      = "mycar-store-us v3.67.0, ios 26.3, SDK 4.10.0"
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

func getWebsocketUri(region string) string {
	switch region {
	case "EMEA":
		return WebsocketUriEMEA
	case "APAC":
		return WebsocketUriAPAC
	case "NORAM":
		return WebsocketUriNORAM
	}
	return WebsocketUriEMEA
}

// wsheaders returns the handshake headers for the VSU websocket. The
// Authorization header carries the RAW access token (no "Bearer" prefix) and
// OUTPUT-FORMAT: PROTO selects the typed protobuf stream.
func wsheaders(accessToken, sessionID, region string) map[string]string {
	risAppVersion := RisWsApplicationVersionEMEA
	appName := XApplicationNameEMEA
	userAgent := WebsocketUserAgent

	switch region {
	case "APAC":
		risAppVersion = RisWsApplicationVersionAPAC
		appName = XApplicationNameAPAC
		userAgent = WebsocketUserAgentAPAC
	case "NORAM":
		risAppVersion = RisWsApplicationVersionNORAM
		appName = XApplicationNameNORAM
		userAgent = WebsocketUserAgentNORAM
	}

	return map[string]string{
		"Authorization":           accessToken,
		"APP-SESSION-ID":          sessionID,
		"OUTPUT-FORMAT":           "PROTO",
		"X-SessionId":             sessionID,
		"X-TrackingId":            uuid.New().String(),
		"ris-os-name":             RisOsName,
		"ris-os-version":          RisOsVersion,
		"ris-sdk-version":         RisWsSdkVersion,
		"ris-application-version": risAppVersion,
		"X-ApplicationName":       appName,
		"X-Locale":                Locale,
		"User-Agent":              userAgent,
	}
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
		headers["User-Agent"] = fmt.Sprintf(UserAgent, RisApplicationVersionNORAM, "ece", RisOsName, RisOsVersion)
	}

	if includeAuthServerHeader {
		headers["Stage"] = "prod"
		headers["X-Device-Id"] = uuid.New().String()
		headers["X-Request-Id"] = uuid.New().String()
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	}

	return headers
}
