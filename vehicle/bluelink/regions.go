package bluelink

import (
	"fmt"
	"strings"
)

// RegionConfig holds the configuration for each brand/region combination
var regionConfigs = map[string]map[string]Config{
	"EU": {
		"hyundai": {
			URI:               "https://prd.eu-ccapi.hyundai.com:8080",
			BasicToken:        "NmQ0NzdjMzgtM2NhNC00Y2YzLTk1NTctMmExOTI5YTk0NjU0OktVeTQ5WHhQekxwTHVvSzB4aEJDNzdXNlZYaG10UVI5aVFobUlGampvWTRJcHhzVg==",
			CCSPServiceID:     "6d477c38-3ca4-4cf3-9557-2a1929a94654",
			CCSPApplicationID: HyundaiAppID,
			AuthClientID:      "64621b96-0f0d-11ec-82a8-0242ac130003",
			BrandAuthUrl:      "https://eu-account.hyundai.com/auth/realms/euhyundaiidm/protocol/openid-connect/auth?client_id=%s&scope=openid+profile+email+phone&response_type=code&hkid_session_reset=true&redirect_uri=%s/api/v1/user/integration/redirect/login&ui_locales=%s&state=%s:%s",
			PushType:          "GCM",
			Cfb:               "RFtoRq/vDXJmRndoZaZQyfOot7OrIqGVFj96iY2WL3yyH5Z/pUvlUhqmCxD2t+D65SQ=",
		},
		"kia": {
			URI:               "https://prd.eu-ccapi.kia.com:8080",
			BasicToken:        "ZmRjODVjMDAtMGEyZi00YzY0LWJjYjQtMmNmYjE1MDA3MzBhOnNlY3JldA==",
			CCSPServiceID:     "fdc85c00-0a2f-4c64-bcb4-2cfb1500730a",
			CCSPApplicationID: KiaAppID,
			AuthClientID:      "572e0304-5f8d-4b4c-9dd5-41aa84eed160",
			BrandAuthUrl:      "https://eu-account.kia.com/auth/realms/eukiaidm/protocol/openid-connect/auth?client_id=%s&scope=openid+profile+email+phone&response_type=code&hkid_session_reset=true&redirect_uri=%s/api/v1/user/integration/redirect/login&ui_locales=%s&state=%s:%s",
			PushType:          "APNS",
			Cfb:               "wLTVxwidmH8CfJYBWSnHD6E0huk0ozdiuygB4hLkM5XCgzAL1Dk5sE36d/bx5PFMbZs=",
		},
	},
	"US": {
		"hyundai": {
			URI:               "https://api.telematics.hyundaiusa.com",
			BasicToken:        "bTY2MTI5QmItZW05My1TUEFIWU4tYlo5MS1hbTQ1NDB6cDE5OTIwOkV2WmdLTElBUGNkdkJabU5OdWtXVGFQN08=",
			CCSPServiceID:     "m66129Bb-em93-SPAHYN-bZ91-am4540zp19920",
			CCSPApplicationID: HyundaiAppID,
			AuthClientID:      "m66129Bb-em93-SPAHYN-bZ91-am4540zp19920",
			BrandAuthUrl:      "https://owners.hyundaiusa.com/us/en/auth/signin",
			PushType:          "GCM",
			Cfb:               "RFtoRq/vDXJmRndoZaZQyfOot7OrIqGVFj96iY2WL3yyH5Z/pUvlUhqmCxD2t+D65SQ=",
		},
		"kia": {
			URI:               "https://api.owners.kia.com",
			BasicToken:        "TXdBTU1PQklMRTo5OGVyLXczNHJmLWliZjMtM2Y2aA==",
			CCSPServiceID:     "MWAMOBILE",
			CCSPApplicationID: KiaAppID,
			AuthClientID:      "MWAMOBILE",
			BrandAuthUrl:      "https://owners.kia.com/us/en/auth/signin",
			PushType:          "APNS",
			Cfb:               "wLTVxwidmH8CfJYBWSnHD6E0huk0ozdiuygB4hLkM5XCgzAL1Dk5sE36d/bx5PFMbZs=",
		},
	},
	"CA": {
		"hyundai": {
			URI:               "https://mybluelink.ca",
			BasicToken:        "SEFUQUhTUEFDQTAyMzIxNDFFRDk3MjJDNjc3MTVBMEI6Q0xJU0NSMDFBSFNQQQ==",
			CCSPServiceID:     "HATAHSPACA0232141ED9722C67715A0B",
			CCSPApplicationID: HyundaiAppID,
			AuthClientID:      "HATAHSPACA0232141ED9722C67715A0B",
			BrandAuthUrl:      "https://mybluelink.ca/tods/api/v2/login",
			PushType:          "GCM",
			Cfb:               "RFtoRq/vDXJmRndoZaZQyfOot7OrIqGVFj96iY2WL3yyH5Z/pUvlUhqmCxD2t+D65SQ=",
		},
		"kia": {
			URI:               "https://kiaconnect.ca",
			BasicToken:        "SEFUQUhTUEFDQTAyMzIxNDFFRDk3MjJDNjc3MTVBMEI6Q0xJU0NSMDFBSFNQQQ==",
			CCSPServiceID:     "HATAHSPACA0232141ED9722C67715A0B",
			CCSPApplicationID: KiaAppID,
			AuthClientID:      "HATAHSPACA0232141ED9722C67715A0B",
			BrandAuthUrl:      "https://kiaconnect.ca/tods/api/v2/login",
			PushType:          "APNS",
			Cfb:               "wLTVxwidmH8CfJYBWSnHD6E0huk0ozdiuygB4hLkM5XCgzAL1Dk5sE36d/bx5PFMbZs=",
		},
	},
	"AU": {
		"kia": {
			URI:               "https://au-apigw.ccs.kia.com.au:8082",
			BasicToken:        "OGFjYjc3OGEtYjkxOC00YThkLTg2MjQtNzNhMGJlYjY0Mjg5OnNlY3JldA==",
			CCSPServiceID:     "8acb778a-b918-4a8d-8624-73a0beb64289",
			CCSPApplicationID: KiaAppID,
			AuthClientID:      "8acb778a-b918-4a8d-8624-73a0beb64289",
			BrandAuthUrl:      "https://au-apigw.ccs.kia.com.au:8082/api/v1/user/oauth2/authorize",
			PushType:          "APNS",
			Cfb:               "wLTVxwidmH8CfJYBWSnHD6E0huk0ozdiuygB4hLkM5XCgzAL1Dk5sE36d/bx5PFMbZs=",
		},
		"hyundai": {
			URI:               "https://au-apigw.ccs.kia.com.au:8082",
			BasicToken:        "OGFjYjc3OGEtYjkxOC00YThkLTg2MjQtNzNhMGJlYjY0Mjg5OnNlY3JldA==",
			CCSPServiceID:     "8acb778a-b918-4a8d-8624-73a0beb64289",
			CCSPApplicationID: HyundaiAppID,
			AuthClientID:      "8acb778a-b918-4a8d-8624-73a0beb64289",
			BrandAuthUrl:      "https://au-apigw.ccs.kia.com.au:8082/api/v1/user/oauth2/authorize",
			PushType:          "GCM",
			Cfb:               "RFtoRq/vDXJmRndoZaZQyfOot7OrIqGVFj96iY2WL3yyH5Z/pUvlUhqmCxD2t+D65SQ=",
		},
	},
}

// GetRegionSettings returns the appropriate configuration for brand and region
func GetRegionSettings(brand, region string) (Config, error) {
	region = strings.ToUpper(region)
	brand = strings.ToLower(brand)

	regions, ok := regionConfigs[region]
	if !ok {
		return Config{}, fmt.Errorf("unsupported region: %s", region)
	}

	config, ok := regions[brand]
	if !ok {
		return Config{}, fmt.Errorf("unsupported brand '%s' for region '%s'", brand, region)
	}

	return config, nil
}
