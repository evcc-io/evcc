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

// Request structure for kamereon api
type Request struct {
	Data Payload `json:"data"`
}

type Payload struct {
	Type       string                 `json:"type"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// Response structure for kamereon api
type Response struct {
	Data struct {
		Type, ID   string     // battery refresh
		Attributes attributes `json:"attributes"`
	} `json:"data"`
	Errors []Error
}

type Error struct {
	Status, Code, Detail string
}

type attributes struct {
	Timestamp          string  `json:"timestamp"`
	ChargingStatus     float32 `json:"chargingStatus"`
	InstantaneousPower int     `json:"instantaneousPower"`
	RangeHvacOff       int     `json:"rangeHvacOff"`    // Nissan
	BatteryAutonomy    int     `json:"batteryAutonomy"` // Renault
	BatteryLevel       int     `json:"batteryLevel"`
	BatteryCapacity    int     `json:"batteryCapacity"` // Nissan
	BatteryTemperature int     `json:"batteryTemperature"`
	PlugStatus         int     `json:"plugStatus"`
	LastUpdateTime     string  `json:"lastUpdateTime"`
	ChargePower        int     `json:"chargePower"`
	RemainingTime      *int    `json:"chargingRemainingTime"`
}
