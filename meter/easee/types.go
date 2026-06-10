package easee

// API is the Easee API endpoint
const API = "https://api.easee.com/api"
const OBSERVATIONS_API = "https://api.easee.com"

// DetectedPowerGridType values
const (
	PowerGridTN3Phase       = 1
	PowerGridTN2PhasePin234 = 2
	PowerGridTN1Phase       = 3
)

// Meter is the meter type“
type Meter struct {
	ID   string
	Name string
}

// Site is the site type
type Site struct {
	ID      int
	SiteKey string
	Name    string
}

type SiteStructure struct {
	RatedCurrent         float64 `json:"ratedCurrent"`
	MaxContinuousCurrent float64 `json:"maxContinuousCurrent"`
	MaxAllocatedCurrent  float64 `json:"maxAllocatedCurrent"`
}
