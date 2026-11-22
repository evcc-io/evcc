package porsche

import "golang.org/x/oauth2"

// Porsche Connect "PPA" app backend, as used by the official My Porsche app.
// Ported from https://github.com/CJNE/pyporscheconnectapi
const (
	ApiURI    = "https://api.ppa.porsche.com/app"
	OAuthURI  = "https://identity.porsche.com"
	Audience  = "https://api.porsche.com"
	ClientID  = "XhygisuebbrqQ80byOuU5VncxLIm8E6H"
	XClientID = "41843fb4-691d-4970-85c7-2673e8ecef40"
	// app user agent, required for current overview data, see CJNE/pyporscheconnectapi@4aad11f
	UserAgent = "de.porsche.one/21.26.46-row+195601 (android)"
	// RedirectURI is the app link registered for the My Porsche app's Auth0 client,
	// see CJNE/pyporscheconnectapi#100. The login runs server-side and reads the
	// code from the redirect location without following it (see login.go).
	RedirectURI = "https://security.porsche.com/auth/en-GB/app/callback"
)

// Scopes requested during authorization (matches the official app).
var Scopes = []string{
	"openid", "profile", "email", "offline_access",
	"mbb", "ssodb", "badge", "vin", "dealers", "cars",
	"charging", "manageCharging", "plugAndCharge",
	"climatisation", "manageClimatisation",
}

// Oauth2Config returns the Auth0 OAuth2 configuration for Porsche Connect.
func Oauth2Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:    ClientID,
		RedirectURL: RedirectURI,
		Endpoint: oauth2.Endpoint{
			AuthURL:   OAuthURI + "/authorize",
			TokenURL:  OAuthURI + "/oauth/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
		Scopes: Scopes,
	}
}

// Measurements requested from the overview endpoint (the subset evcc needs).
var Measurements = []string{
	"BATTERY_LEVEL",
	"E_RANGE",
	"RANGE",
	"CHARGING_SUMMARY",
	"CHARGING_RATE",
	"HVAC_SUMMARY",
	"MILEAGE",
	"GPS_LOCATION",
}
