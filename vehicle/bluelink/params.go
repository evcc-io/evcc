package bluelink

const (
	KiaAppID     = "a2b8469b-30a3-4361-8e13-6fceea8fbe74"
	HyundaiAppID = "014d2225-8495-4735-812d-2616334fd15d"
	GenesisAppID = "f11f2b86-e0e7-4851-90df-5600b01d8b70"
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
	BrandKia     = "kia"
	BrandHyundai = "hyundai"
	BrandGenesis = "genesis"
)
