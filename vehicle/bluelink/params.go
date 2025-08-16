package bluelink

import (
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// Config is the bluelink API configuration
type BluelinkConfig struct {
	URI               string
	AuthClientID      string // v2
	BrandAuthUrl      string // v2
	BasicToken        string
	CCSPServiceID     string
	CCSPApplicationID string
	PushType          string
	Cfb               string
	LoginFormHost     string
}

// Identity implements the Kia/Hyundai bluelink identity.
// Based on https://github.com/Hacksore/bluelinky.
type Identity struct {
	*request.Helper
	log      *util.Logger
	config   BluelinkConfig
	deviceID string
	oauth2.TokenSource
}

const (
	DeviceIdURL        = "/api/v1/spa/notifications/register"
	IntegrationInfoURL = "/api/v1/user/integrationinfo"
	SilentSigninURL    = "/api/v1/user/silentsignin"
	LanguageURL        = "/api/v1/user/language"
	LoginURL           = "/api/v1/user/signin"
	TokenURL           = "/api/v1/user/oauth2/token"
)

// central auth configuration parameters
var (
	// region -> brand -> settings
	ConfigMap = map[string]map[string]map[string]string{
		RegionAustralia: {
			BrandHyundai: {
				"AppId":      "f9ccfdac-a48d-4c57-bd32-9116963c24ed",
				"BasicToken": "ODU1YzcyZGYtZGZkNy00MjMwLWFiMDMtNjdjYmY5MDJiYjFjOmU2ZmJ3SE0zMllOYmhRbDBwdmlhUHAzcmY0dDNTNms5MWVjZUEzTUpMZGJkVGhDTw==",
				"Cfb":        "nGDHng3k4Cg9gWV+C+A6Yk/ecDopUNTkGmDpr2qVKAQXx9bvY2/YLoHPfObliK32mZQ=",
				"ServiceId":  "855c72df-dfd7-4230-ab03-67cbf902bb1c",
				"URI":        "https://au-apigw.ccs.hyundai.com.au:8080",
			},
			BrandKia: {
				"AppId":      "4ad4dcde-be23-48a8-bc1c-91b94f5c06f8",
				"BasicToken": "OGFjYjc3OGEtYjkxOC00YThkLTg2MjQtNzNhMGJlYjY0Mjg5OjdTY01NbTZmRVlYZGlFUEN4YVBhUW1nZVlkbFVyZndvaDRBZlhHT3pZSVMyQ3U5VA==",
				"Cfb":        "SGGCDRvrzmRa2WTNFQPUaNfSFdtPklZ48xUuVckigYasxmeOQqVgCAC++YNrI1vVabI=",
				"ServiceId":  "8acb778a-b918-4a8d-8624-73a0beb64289",
				"URI":        "https://au-apigw.ccs.kia.com.au:8082",
			},
		},
		RegionEurope: {
			BrandGenesis: {
				"AppId":         "f11f2b86-e0e7-4851-90df-5600b01d8b70",
				"AuthClientId":  "3020afa2-30ff-412a-aa51-d28fbe901e10",
				"BasicToken":    "MzAyMGFmYTItMzBmZi00MTJhLWFhNTEtZDI4ZmJlOTAxZTEwOkZLRGRsZWYyZmZkbGVGRXdlRUxGS0VSaUxFUjJGRUQyMXNEZHdkZ1F6NmhGRVNFMw==",
				"BrandAuthUrl":  "%s/auth/api/v2/user/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s/api/v1/user/oauth2/redirect&lang=%s&state=ccsp",
				"Cfb":           "RFtoRq/vDXJmRndoZaZQyYo3/qFLtVReW8P7utRPcc0ZxOzOELm9mexvviBk/qqIp4A=",
				"LoginFormHost": "https://accounts-eu.genesis.com",
				"PushType":      "GCM",
				"ServiceId":     "3020afa2-30ff-412a-aa51-d28fbe901e10",
				"URI":           "https://prd.eu-ccapi.genesis.com:443",
			},
			BrandHyundai: {
				"AppId":         "014d2225-8495-4735-812d-2616334fd15d",
				"AuthClientId":  "64621b96-0f0d-11ec-82a8-0242ac130003",
				"BasicToken":    "NmQ0NzdjMzgtM2NhNC00Y2YzLTk1NTctMmExOTI5YTk0NjU0OktVeTQ5WHhQekxwTHVvSzB4aEJDNzdXNlZYaG10UVI5aVFobUlGampvWTRJcHhzVg==",
				"BrandAuthUrl":  "https://eu-account.hyundai.com/auth/realms/euhyundaiidm/protocol/openid-connect/auth?client_id=%s&scope=openid+profile+email+phone&response_type=code&hkid_session_reset=true&redirect_uri=%s/api/v1/user/integration/redirect/login&ui_locales=%s&state=%s:%s",
				"Cfb":           "RFtoRq/vDXJmRndoZaZQyfOot7OrIqGVFj96iY2WL3yyH5Z/pUvlUhqmCxD2t+D65SQ=",
				"LoginFormHost": "",
				"PushType":      "GCM",
				"ServiceId":     "6d477c38-3ca4-4cf3-9557-2a1929a94654",
				"URI":           "https://prd.eu-ccapi.hyundai.com:8080",
			},
			BrandKia: {
				"AppId":         "a2b8469b-30a3-4361-8e13-6fceea8fbe74",
				"AuthClientId":  "fdc85c00-0a2f-4c64-bcb4-2cfb1500730a",
				"BasicToken":    "ZmRjODVjMDAtMGEyZi00YzY0LWJjYjQtMmNmYjE1MDA3MzBhOnNlY3JldA==",
				"BrandAuthUrl":  "%s/auth/api/v2/user/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s/api/v1/user/oauth2/redirect&lang=%s&state=ccsp",
				"Cfb":           "wLTVxwidmH8CfJYBWSnHD6E0huk0ozdiuygB4hLkM5XCgzAL1Dk5sE36d/bx5PFMbZs=",
				"LoginFormHost": "https://idpconnect-eu.kia.com",
				"PushType":      "APNS",
				"ServiceId":     "fdc85c00-0a2f-4c64-bcb4-2cfb1500730a",
				"URI":           "https://prd.eu-ccapi.kia.com:8080",
			},
		},
		RegionNewZealand: {
			// Hyundai NZ seems to be identical to Hyundai AU
			BrandHyundai: {
				"AppId":      "f9ccfdac-a48d-4c57-bd32-9116963c24ed",
				"BasicToken": "ODU1YzcyZGYtZGZkNy00MjMwLWFiMDMtNjdjYmY5MDJiYjFjOmU2ZmJ3SE0zMllOYmhRbDBwdmlhUHAzcmY0dDNTNms5MWVjZUEzTUpMZGJkVGhDTw==",
				"Cfb":        "nGDHng3k4Cg9gWV+C+A6Yk/ecDopUNTkGmDpr2qVKAQXx9bvY2/YLoHPfObliK32mZQ=",
				"ServiceId":  "855c72df-dfd7-4230-ab03-67cbf902bb1c",
				"URI":        "https://au-apigw.ccs.hyundai.com.au:8080",
			},
			BrandKia: {
				"AppId":      "97745337-cac6-4a5b-afc3-e65ace81c994",
				"BasicToken": "NGFiNjA2YTctY2VhNC00OGEwLWEyMTYtZWQ5YzE0YTRhMzhjOjBoYUZxWFRrS2t0Tktmemt4aFowYWt1MzFpNzRnMHlRRm01b2QybXo0TGRJNW1MWQ==",
				"Cfb":        "SGGCDRvrzmRa2WTNFQPUaC1OsnAhQgPgcQETEfbY8abEjR/ICXK0p+Rayw5tHCGyiUA=",
				"ServiceId":  "4ab606a7-cea4-48a0-a216-ed9c14a4a38c",
				"URI":        "https://au-apigw.ccs.kia.com.au:8082",
			},
		},
	}
)

// as returned by web configurator
// since other templates already use the ISO 3166-A2 codes (hopefully) we use those, too
const (
	RegionEurope     = "EU" // OAuth2
	RegionCanada     = "CA" // legacy
	RegionUSA        = "US" // legacy + different method for Kia / Hyundai
	RegionChina      = "CN" // OAuth2
	RegionAustralia  = "AU" // OAuth2
	RegionIndia      = "IN" // OAuth2
	RegionNewZealand = "NZ" // legacy + no Hyundai
)

const (
	CAClientID     = "HATAHSPACA0232141ED9722C"
	CAClientSecret = "CLISCR01AHSPA"
	CADeviceID     = "TW96aWxsYS81LjAgKFdpbmRvd3MgTlQgMTAuMDsgV2luNjQ7IHg2NCkgQXBwbGVXZWJLaXQvNTM3LjM2IChLSFRNTCwgbGlrZSBHZWNrbykgQ2hyb21lLzEzOC4wLjAuMCBTYWZhcmkvNTM3LjM2IEVkZy8xMzguMC4wLjArV2luMzIrMTIzNCsxMjM0"
)

const (
	BrandKia     = "kia"
	BrandHyundai = "hyundai"
	BrandGenesis = "genesis"
)
