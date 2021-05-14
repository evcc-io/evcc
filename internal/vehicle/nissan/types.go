package nissan

// api constants
const (
	APIVersion         = "protocol=1.0,resource=2.1"
	ClientID           = "a-ncb-prod-android"
	ClientSecret       = "3LBs0yOx2XO-3m4mMRW27rKeJzskhfWF0A8KUtnim8i/qYQPl8ZItp3IaqJXaYj_"
	Scope              = "openid profile vehicles"
	AuthBaseURL        = "https://prod.eu.auth.kamereon.org/kauth"
	Realm              = "a-ncb-prod"
	RedirectURI        = "org.kamereon.service.nci:/oauth2redirect"
	CarAdapterBaseURL  = "https://alliance-platform-caradapter-prod.apps.eu.kamereon.io/car-adapter"
	UserAdapterBaseURL = "https://alliance-platform-usersadapter-prod.apps.eu.kamereon.io/user-adapter"
	UserBaseURL        = "https://nci-bff-web-prod.apps.eu.kamereon.io/bff-web"
)

const timeFormat = "2006-01-02T15:04:05Z"

type Auth struct {
	AuthID    string         `json:"authId"`
	Template  string         `json:"template"`
	Stage     string         `json:"stage"`
	Header    string         `json:"header"`
	Callbacks []AuthCallback `json:"callbacks"`
}

type AuthCallback struct {
	Type   string              `json:"type"`
	Output []AuthCallbackValue `json:"output"`
	Input  []AuthCallbackValue `json:"input"`
}

type AuthCallbackValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Token struct {
	TokenID    string `json:"tokenId"`
	SuccessURL string `json:"successUrl"`
	Realm      string `json:"realm"`
}

type Vehicles struct {
	Data []Vehicle
}

type Vehicle struct {
	VIN        string
	ModelName  string
	PictureURL string
}
