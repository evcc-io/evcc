// Native Go implementation for Braiins OS evcc integration
//
// REQUIREMENTS:
// - Braiins OS with API enabled (default port 80)
// - For dynamic power control: Enable "Power Target" mode in Braiins OS tuner settings
// - Without Power Target: Only on/off control available
//
// Version: 1.4.2 (Fixed: negative current validation, resource leak prevention, bot suggestions)
// Tested with real API v1.0.0
// https://developer.braiins-os.com/latest/openapi.html

package charger

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Miner status constants
const (
	MinerStatusMining = 2 // Mining active
	MinerStatusPaused = 3 // Mining paused
)

// API endpoints
const (
	apiPathLogin        = "/api/v1/auth/login"
	apiPathMinerDetails = "/api/v1/miner/details"
	apiPathMinerStats   = "/api/v1/miner/stats"
	apiPathPause        = "/api/v1/actions/pause"
	apiPathResume       = "/api/v1/actions/resume"
	apiPathConstraints  = "/api/v1/configuration/constraints"
	apiPathPerformance  = "/api/v1/performance/mode"
	apiPathPowerTarget  = "/api/v1/performance/power-target"
)

// BraiinsOS charger implementation
type BraiinsOS struct {
	*request.Helper
	*embed
	uri            string
	user           string
	password       string
	configMaxPower int
	voltage        float64 // Configurable grid voltage

	// Configurable rate limiting and stepping parameters
	powerTargetInterval time.Duration // Minimum interval between power target updates
	powerTargetStep     int           // Power target stepping in watts

	// Hardware constraints discovered from miner
	minWatts     int
	defaultWatts int
	maxWatts     int

	// Power target capability and warning state
	powerTargetEnabled bool
	powerTargetWarned  bool // To avoid spam warnings

	// Thread-safe fields protected by mutex
	mu              sync.Mutex
	token           string
	tokenExpiry     time.Time
	lastPowerUpdate time.Time // Last power target update timestamp
	lastPowerTarget int       // Last set power target for comparison

	log *util.Logger
}

// BraiinsConfig is the configuration struct
type BraiinsConfig struct {
	URI                 string        `mapstructure:"uri"`
	User                string        `mapstructure:"user"`
	Password            string        `mapstructure:"password"`
	Timeout             time.Duration `mapstructure:"timeout"`
	MaxPower            int           `mapstructure:"maxPower"`            // Optional: User-defined power limit
	Voltage             float64       `mapstructure:"voltage"`             // Configurable grid voltage
	PowerTargetInterval time.Duration `mapstructure:"powerTargetInterval"` // Configurable rate limiting interval
	PowerTargetStep     int           `mapstructure:"powerTargetStep"`     // Configurable power stepping
}

// Login request/response structures
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token    string `json:"token"`
	TimeoutS int    `json:"timeout_s"`
}

// MinerDetails for status detection
type MinerDetails struct {
	Status int `json:"status"`
}

// MinerStats for power measurement
type MinerStats struct {
	PowerStats struct {
		ApproximatedConsumption struct {
			Watt int `json:"watt"`
		} `json:"approximated_consumption"`
	} `json:"power_stats"`
}

// PowerTarget structures
type PowerTarget struct {
	Watt int `json:"watt"`
}

// ConfigConstraints for power limits discovery
type ConfigConstraints struct {
	TunerConstraints struct {
		PowerTarget struct {
			Min struct {
				Watt int `json:"watt"`
			} `json:"min"`
			Default struct {
				Watt int `json:"watt"`
			} `json:"default"`
			Max struct {
				Watt int `json:"watt"`
			} `json:"max"`
		} `json:"power_target"`
	} `json:"tuner_constraints"`
}

func init() {
	registry.Add("braiins", NewBraiinsFromConfig)
}

// NewBraiinsFromConfig creates a Braiins charger from generic config
func NewBraiinsFromConfig(other map[string]interface{}) (api.Charger, error) {
	var cc BraiinsConfig
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// Set defaults for missing configuration values
	if cc.Timeout == 0 {
		cc.Timeout = 15 * time.Second
	}
	if cc.User == "" {
		cc.User = "root"
	}
	if cc.Voltage == 0 {
		cc.Voltage = 230.0 // Default: Europe standard
	}
	if cc.PowerTargetInterval == 0 {
		cc.PowerTargetInterval = 30 * time.Second // Default: 30 seconds
	}
	if cc.PowerTargetStep == 0 {
		cc.PowerTargetStep = 100 // Default: 100 watts
	}

	uri := fmt.Sprintf("http://%s", cc.URI)
	return NewBraiins(uri, cc.User, cc.Password, cc.Timeout, cc.MaxPower, cc.Voltage, cc.PowerTargetInterval, cc.PowerTargetStep)
}

// NewBraiins creates Braiins charger
func NewBraiins(uri, user, password string, timeout time.Duration, maxPower int, voltage float64, powerTargetInterval time.Duration, powerTargetStep int) (api.Charger, error) {
	log := util.NewLogger("braiins")

	c := &BraiinsOS{
		Helper: request.NewHelper(log),
		embed: &embed{
			Icon_:     "generic",
			Features_: []api.Feature{api.IntegratedDevice},
		},
		log:                 log,
		uri:                 uri,
		user:                user,
		password:            password,
		configMaxPower:      maxPower,
		voltage:             voltage,
		powerTargetInterval: powerTargetInterval,
		powerTargetStep:     powerTargetStep,
	}

	c.Client.Timeout = timeout

	// Test connection and get initial token
	if err := c.login(); err != nil {
		c.log.ERROR.Printf("Connection test failed: %v", err)
		return nil, fmt.Errorf("connection test failed: %w", err)
	}

	// Discover miner constraints and power target capability
	if err := c.discoverConstraints(); err != nil {
		c.log.ERROR.Printf("Failed to get miner constraints: %v", err)
		return nil, fmt.Errorf("failed to get miner constraints: %w", err)
	}

	// Log configuration summary with complete hardware range information
	effectiveMax := c.getEffectiveMaxPower()
	if c.powerTargetEnabled {
		var maxLabel string
		if c.configMaxPower > 0 {
			maxLabel = "User"
		} else {
			maxLabel = "Default"
		}

		c.log.INFO.Printf("Braiins miner ready at %s with power control - evcc: %dW (Min.) - %dW (%s), hardware: %dW (Min.) - %dW (Default) - %dW (Max.), %.0fV, interval: %v, step: %dW",
			uri, c.minWatts, effectiveMax, maxLabel, c.minWatts, c.defaultWatts, c.maxWatts, c.voltage, c.powerTargetInterval, c.powerTargetStep)
	} else {
		c.log.INFO.Printf("Braiins miner ready at %s with on/off control (%.0fV)", uri, c.voltage)
	}

	return c, nil
}

// login gets a new authentication token with thread-safe token management
func (c *BraiinsOS) login() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if time.Now().Before(c.tokenExpiry) && c.token != "" {
		return nil // Token still valid
	}

	loginReq := LoginRequest{
		Username: c.user,
		Password: c.password,
	}

	req, err := request.New(http.MethodPost, c.uri+apiPathLogin, request.MarshalJSON(loginReq), request.JSONEncoding)
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	var resp LoginResponse
	if err := c.DoJSON(req, &resp); err != nil {
		c.log.ERROR.Printf("Login request failed: %v", err)
		return fmt.Errorf("login request failed: %w", err)
	}

	if resp.Token == "" {
		c.log.ERROR.Printf("Login succeeded, but no token was received.")
		return fmt.Errorf("no token received")
	}

	c.token = resp.Token

	// Set token expiry with safety buffer
	tokenTimeout := time.Duration(resp.TimeoutS) * time.Second
	if tokenTimeout <= 0 {
		tokenTimeout = 1 * time.Hour
	}
	if tokenTimeout > 30*time.Second {
		c.tokenExpiry = time.Now().Add(tokenTimeout - 30*time.Second)
	} else {
		c.tokenExpiry = time.Now().Add(tokenTimeout)
	}

	c.log.DEBUG.Printf("Login successful, token expires in %s", tokenTimeout)
	return nil
}

// authRequest makes an authenticated HTTP request with automatic retry on 401
func (c *BraiinsOS) authRequest(method, path string, body any) (*http.Response, error) {
	// First attempt with current token
	resp, err := c.doAuthenticatedRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	// If 401 Unauthorized: invalidate token and retry once
	if resp.StatusCode == http.StatusUnauthorized {
		c.log.DEBUG.Printf("Token invalid (401), attempting re-authentication")

		// Close the first response
		resp.Body.Close()

		// Invalidate current token to force new login
		c.mu.Lock()
		c.token = ""
		c.tokenExpiry = time.Time{}
		c.mu.Unlock()

		// Retry with fresh token - handle potential second 401 to prevent resource leak
		retryResp, retryErr := c.doAuthenticatedRequest(method, path, body)
		if retryErr != nil {
			return nil, retryErr
		}

		// Ensure response body is closed if second attempt also fails auth
		if retryResp.StatusCode == http.StatusUnauthorized {
			retryResp.Body.Close()
			return nil, fmt.Errorf("authentication failed after retry, token refresh unsuccessful")
		}

		return retryResp, nil
	}

	return resp, nil
}

// doAuthenticatedRequest performs the actual HTTP request with current token
func (c *BraiinsOS) doAuthenticatedRequest(method, path string, body any) (*http.Response, error) {
	if err := c.login(); err != nil {
		return nil, err
	}

	var req *http.Request
	var err error

	if body != nil {
		req, err = request.New(method, c.uri+path, request.MarshalJSON(body), request.JSONEncoding)
	} else {
		req, err = request.New(method, c.uri+path, nil, request.JSONEncoding)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create authenticated request: %w", err)
	}

	// Get token in thread-safe manner
	c.mu.Lock()
	token := c.token
	c.mu.Unlock()

	req.Header.Set("Authorization", token)
	return c.Do(req)
}

// handleHTTPResponse checks status codes and provides consistent error handling
func (c *BraiinsOS) handleHTTPResponse(resp *http.Response, operation string) error {
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("authentication failed after retry, token invalid: %s (HTTP %d)", resp.Status, resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil // 204 No Content is success for PUT operations
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s failed: %s (HTTP %d)", operation, resp.Status, resp.StatusCode)
	}
	return nil
}

// closeResponseBody safely closes response body with error logging
func (c *BraiinsOS) closeResponseBody(resp *http.Response) {
	if err := resp.Body.Close(); err != nil {
		c.log.DEBUG.Printf("Failed to close response body: %v", err)
	}
}

// discoverConstraints gets miner power limits from API and detects power target capability
func (c *BraiinsOS) discoverConstraints() error {
	resp, err := c.authRequest(http.MethodGet, apiPathConstraints, nil)
	if err != nil {
		return fmt.Errorf("constraints request failed: %w", err)
	}
	defer c.closeResponseBody(resp)

	if err := c.handleHTTPResponse(resp, "constraints request"); err != nil {
		return err
	}

	var constraints ConfigConstraints
	if err := json.NewDecoder(resp.Body).Decode(&constraints); err != nil {
		return fmt.Errorf("failed to decode constraints: %w", err)
	}

	c.minWatts = constraints.TunerConstraints.PowerTarget.Min.Watt
	c.defaultWatts = constraints.TunerConstraints.PowerTarget.Default.Watt
	c.maxWatts = constraints.TunerConstraints.PowerTarget.Max.Watt

	// Detect power target support based on valid constraint values
	c.powerTargetEnabled = c.minWatts > 0 && c.defaultWatts > 0 && c.maxWatts > 0

	c.log.DEBUG.Printf("Discovered power constraints: min=%dW, default=%dW, max=%dW, powerTargetEnabled=%v",
		c.minWatts, c.defaultWatts, c.maxWatts, c.powerTargetEnabled)

	return nil
}

// getEffectiveMaxPower returns the effective maximum power for evcc control
func (c *BraiinsOS) getEffectiveMaxPower() int {
	// Default to miner's reported default power
	effectiveMax := c.defaultWatts

	// Use user-configured value if provided
	if c.configMaxPower > 0 {
		effectiveMax = c.configMaxPower
	}

	// Constrain to hardware limits
	effectiveMax = int(math.Min(float64(effectiveMax), float64(c.maxWatts)))
	effectiveMax = int(math.Max(float64(effectiveMax), float64(c.minWatts)))

	return effectiveMax
}

// getMinerStatus gets the current miner status from API
func (c *BraiinsOS) getMinerStatus() (int, error) {
	resp, err := c.authRequest(http.MethodGet, apiPathMinerDetails, nil)
	if err != nil {
		return 0, fmt.Errorf("miner details request failed: %w", err)
	}
	defer c.closeResponseBody(resp)

	if err := c.handleHTTPResponse(resp, "miner details"); err != nil {
		return 0, err
	}

	var details MinerDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return 0, fmt.Errorf("failed to decode miner details: %w", err)
	}

	return details.Status, nil
}

// setPowerTarget sets the miner power target with thread-safe tracking
func (c *BraiinsOS) setPowerTarget(targetWatts int) error {
	resp, err := c.authRequest(http.MethodPut, apiPathPowerTarget, PowerTarget{Watt: targetWatts})
	if err != nil {
		return fmt.Errorf("set power target failed: %w", err)
	}
	defer c.closeResponseBody(resp)

	if err := c.handleHTTPResponse(resp, "set power target"); err != nil {
		return err
	}

	// Update tracking variables in thread-safe manner
	c.mu.Lock()
	c.lastPowerTarget = targetWatts
	c.lastPowerUpdate = time.Now()
	c.mu.Unlock()

	c.log.DEBUG.Printf("Power target set to %dW", targetWatts)
	return nil
}

// Status implements the api.Charger interface
func (c *BraiinsOS) Status() (api.ChargeStatus, error) {
	status, err := c.getMinerStatus()
	if err != nil {
		return api.StatusNone, err
	}

	// Map miner status to charger status
	switch status {
	case MinerStatusMining:
		return api.StatusC, nil // Mining active
	case MinerStatusPaused:
		return api.StatusB, nil // Mining paused
	default:
		return api.StatusNone, nil // Unknown status
	}
}

// Enabled implements the api.Charger interface
func (c *BraiinsOS) Enabled() (bool, error) {
	status, err := c.getMinerStatus()
	if err != nil {
		return false, err
	}
	// Only enabled when actively mining (not when paused)
	return status == MinerStatusMining, nil
}

// Enable implements the api.Charger interface
func (c *BraiinsOS) Enable(enable bool) error {
	endpoint := apiPathPause
	operation := "pause"
	if enable {
		endpoint = apiPathResume
		operation = "resume"
	}

	resp, err := c.authRequest(http.MethodPut, endpoint, nil)
	if err != nil {
		return err
	}
	defer c.closeResponseBody(resp)

	if err := c.handleHTTPResponse(resp, operation); err != nil {
		return err
	}

	c.log.DEBUG.Printf("Miner %s successful", operation)
	return nil
}

// MaxCurrent implements the api.Charger interface with configurable voltage and power control
func (c *BraiinsOS) MaxCurrent(current int64) error {
	// Validate input - reject negative current values
	if current < 0 {
		c.log.WARN.Printf("Received invalid negative current value: %dA - rejecting request", current)
		return fmt.Errorf("invalid negative current value: %d", current)
	}

	if current == 0 {
		return c.Enable(false) // Pause mining
	}

	// Calculate desired power based on current amperage and configured voltage
	powerRequest := float64(current) * c.voltage

	// Check if requested power meets minimum hardware requirements for PV surplus operation
	if powerRequest < float64(c.minWatts) {
		c.log.DEBUG.Printf("Requested %.1fA (%.0fW) insufficient for hardware minimum (%dW) - keeping miner paused",
			float64(current), powerRequest, c.minWatts)
		return c.Enable(false) // Pause - insufficient PV surplus for hardware minimum
	}

	// Graceful fallback for miners not in power target mode
	if !c.powerTargetEnabled {
		// Simply enable without power control
		if err := c.Enable(true); err != nil {
			return err
		}
		// Only warn once to avoid spam
		if !c.powerTargetWarned {
			c.log.WARN.Printf("Enable Power Target in Braiins OS for dynamic power control")
			c.powerTargetWarned = true
		}
		return nil // Simple on/off, no power control
	}

	effectiveMax := c.getEffectiveMaxPower()
	if effectiveMax <= c.minWatts {
		c.log.WARN.Printf("Effective max power (%dW) too low for dynamic control - using minimum (%dW)", effectiveMax, c.minWatts)
		// Set power target first, then enable to avoid power spikes
		if err := c.setPowerTarget(c.minWatts); err != nil {
			return err
		}
		return c.Enable(true)
	}

	// Apply power limits with explicit rounding
	targetPower := math.Max(float64(c.minWatts), powerRequest)
	targetPower = math.Min(float64(effectiveMax), targetPower)

	// Round down using configurable power step to avoid grid consumption
	targetPower = math.Floor(targetPower/float64(c.powerTargetStep)) * float64(c.powerTargetStep)
	targetPowerInt := int(targetPower)

	// Rate limiting: only update if significant change or enough time passed
	c.mu.Lock()
	timeSinceLastUpdate := time.Since(c.lastPowerUpdate)
	powerChange := targetPowerInt != c.lastPowerTarget
	lastTarget := c.lastPowerTarget
	c.mu.Unlock()

	if !powerChange {
		c.log.DEBUG.Printf("Power target unchanged at %dW, skipping update", lastTarget)
		return nil
	}

	// Use configurable rate limiting interval
	if timeSinceLastUpdate < c.powerTargetInterval {
		c.log.DEBUG.Printf("Rate limiting: %.0fs since last update, delaying power change to %dW",
			timeSinceLastUpdate.Seconds(), targetPowerInt)
		return nil
	}

	c.log.DEBUG.Printf("Requested %.1fA at %.0fV, setting power target to %dW (rounded down from %.0fW)",
		float64(current), c.voltage, targetPowerInt, powerRequest)

	// Set power target before enabling to avoid power spikes
	if err := c.setPowerTarget(targetPowerInt); err != nil {
		return err
	}

	// Then enable the miner at the desired power level
	return c.Enable(true)
}

// CurrentPower implements the api.Meter interface
func (c *BraiinsOS) CurrentPower() (float64, error) {
	resp, err := c.authRequest(http.MethodGet, apiPathMinerStats, nil)
	if err != nil {
		return 0, fmt.Errorf("stats request failed: %w", err)
	}
	defer c.closeResponseBody(resp)

	if err := c.handleHTTPResponse(resp, "stats"); err != nil {
		return 0, err
	}

	var stats MinerStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return 0, fmt.Errorf("failed to decode miner stats: %w", err)
	}

	power := float64(stats.PowerStats.ApproximatedConsumption.Watt)
	c.log.DEBUG.Printf("Current power consumption: %.0fW", power)

	return power, nil
}

// Currents implements the api.PhaseCurrents interface with configurable voltage
func (c *BraiinsOS) Currents() (float64, float64, float64, error) {
	power, err := c.CurrentPower()
	if err != nil {
		return 0, 0, 0, err
	}

	// Calculate current using configured voltage
	current := power / c.voltage
	return current, 0, 0, nil
}

// Interface compliance checks
var _ api.Charger = (*BraiinsOS)(nil)
var _ api.Meter = (*BraiinsOS)(nil)
var _ api.PhaseCurrents = (*BraiinsOS)(nil)
