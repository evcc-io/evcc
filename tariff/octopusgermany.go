package tariff

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

const (
	// Token management constants (from Python implementation)
	tokenRefreshMargin   = 5 * time.Minute  // Refresh token if less than 5 minutes remaining
	tokenAutoRefreshTime = 50 * time.Minute // Auto refresh token every 50 minutes
	apiRetryDelay        = 2 * time.Second  // Delay between retries
	maxTokenRetries      = 3                // Maximum token refresh retries
)

var (
	berlinLocation  *time.Location
	startupDetector *StartupDetector
)

// StartupDetector manages vehicle integration startup for cached instances
type StartupDetector struct {
	mu        sync.Mutex
	instances []*OctopusGermany
	started   bool
}

// NewStartupDetector creates a new startup detector
func NewStartupDetector() *StartupDetector {
	return &StartupDetector{
		instances: make([]*OctopusGermany, 0),
	}
}

// Register adds an OctopusGermany instance to be monitored for startup
func (sd *StartupDetector) Register(instance *OctopusGermany) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.instances = append(sd.instances, instance)

	// If we've already detected startup, immediately trigger vehicle integration
	if sd.started {
		go instance.startVehicleIntegrationIfNeeded()
	}
}

// DetectStartup triggers vehicle integration for all registered instances
func (sd *StartupDetector) DetectStartup() {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	if sd.started {
		return // Already started
	}

	sd.started = true

	// Start vehicle integration for all registered instances
	for _, instance := range sd.instances {
		go instance.startVehicleIntegrationIfNeeded()
	}
}

func init() {
	var err error
	berlinLocation, err = time.LoadLocation("Europe/Berlin")
	if err != nil {
		berlinLocation = time.Local
	}

	// Initialize global startup detector
	startupDetector = NewStartupDetector()

	// Start a global vehicle integration service that works independently of tariff instances
	go func() {
		// Wait for EVCC to initialize
		time.Sleep(5 * time.Second)

		// Start the standalone vehicle integration service
		startStandaloneVehicleIntegration()
	}()

	registry.AddCtx("octopusgermany", NewOctopusGermanyFromConfig)
}

// Config holds the configuration for the OctopusGermany adapter.
type Config struct {
	Email         string `mapstructure:"email"`         // Octopus Germany account email (required)
	Password      string `mapstructure:"password"`      // Octopus Germany account password (required)
	ProductCode   string `mapstructure:"productcode"`   // Product code for the tariff (optional, auto-discovered if not set)
	AccountNumber string `mapstructure:"accountnumber"` // Account number (optional, auto-selected if only one account available)
	Vehicle       string `mapstructure:"vehicle"`       // Vehicle name for charging plan integration (optional, only for DEU-ELECTRICITY-IO-GO-24)
}

// OctopusGermany holds the configuration and state for the Octopus Germany tariff.
type OctopusGermany struct {
	log    *util.Logger
	config Config
	*request.Helper

	tokenMu                   sync.RWMutex
	token                     string
	tokenExp                  time.Time
	data                      *util.Monitor[api.Rates]
	vehicleIntegrationStarted bool      // Flag to ensure vehicle integration is only started once
	lastVehiclePlanUpdate     time.Time // Track when we last updated the vehicle plan
	startupTime               time.Time // Track when this instance was created
	lastRatesCall             time.Time // Track when Rates() was last called
	vehicleIntegrationChecked bool      // Flag to ensure vehicle integration check happens only once per restart
}

var _ api.Tariff = (*OctopusGermany)(nil)

// NewOctopusGermanyFromConfig creates a new OctopusGermany tariff from configuration.
func NewOctopusGermanyFromConfig(ctx context.Context, other map[string]interface{}) (api.Tariff, error) {
	var config Config
	if err := util.DecodeOther(other, &config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	log := util.NewLogger("octopusgermany")

	// Validate required fields
	if config.Email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if config.Password == "" {
		return nil, fmt.Errorf("password is required")
	}

	octopus := &OctopusGermany{
		log:         log,
		config:      config,
		Helper:      request.NewHelper(log),
		data:        util.NewMonitor[api.Rates](2 * time.Hour),
		startupTime: time.Now(), // Record when this instance was created
	}

	// Login first to access API
	if err := octopus.login(); err != nil {
		return nil, fmt.Errorf("initial login failed: %w", err)
	}

	// Auto-discover account if not provided
	if config.AccountNumber == "" {
		log.INFO.Println("AccountNumber not set - running discovery to find and auto-select account")

		if results, err := octopus.discovery(); err != nil {
			log.INFO.Printf("Discovery failed: %v", err)
			return nil, fmt.Errorf("discovery failed: %w", err)
		} else {
			log.INFO.Printf("Discovery found %d accounts", len(results))

			if len(results) == 0 {
				return nil, fmt.Errorf("no accounts found for this user")
			}

			// Show all available accounts and their products
			for i, result := range results {
				log.INFO.Printf("  %d. Account: %s, Products: %v", i+1, result.AccountNumber, result.ProductCodes)
			}

			if len(results) == 1 {
				// Auto-select the single account
				octopus.config.AccountNumber = results[0].AccountNumber
				log.INFO.Printf("Auto-selected single account: %s", octopus.config.AccountNumber)
			} else {
				// Multiple accounts found - user must choose
				log.INFO.Printf("Multiple accounts found - please specify accountnumber in config")
				return nil, fmt.Errorf("multiple accounts found (%d) - please specify accountnumber in config", len(results))
			}
		}
	}

	// Auto-discover and select product if not provided
	if config.ProductCode == "" {
		log.INFO.Println("ProductCode not set - running discovery to find and auto-select active tariff")

		// Get available products for the account (use the potentially auto-selected account number)
		availableProducts, err := octopus.discoverAccountProducts(octopus.config.AccountNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to discover products for account %s: %w", octopus.config.AccountNumber, err)
		}

		if len(availableProducts) == 0 {
			return nil, fmt.Errorf("no products found for account %s", octopus.config.AccountNumber)
		}

		// Log all available products
		log.INFO.Printf("Available products for account %s:", octopus.config.AccountNumber)
		for i, product := range availableProducts {
			log.INFO.Printf("  %d. %s", i+1, product.Code)
		}

		// Auto-select the first active product (most recent/current)
		selectedProduct := availableProducts[0].Code
		octopus.config.ProductCode = selectedProduct

		log.INFO.Printf("Auto-selected active product: %s", selectedProduct)
	}

	// Log startup with final configuration (v4 with enhanced dispatches processing)
	if octopus.config.Vehicle != "" && octopus.config.ProductCode == "DEU-ELECTRICITY-IO-GO-24" {
		log.INFO.Printf("OctopusGermany initialized: Account=%s, Product=%s, Vehicle=%s (charging optimization enabled)",
			octopus.config.AccountNumber, octopus.config.ProductCode, octopus.config.Vehicle)
	} else {
		log.INFO.Printf("OctopusGermany initialized: Account=%s, Product=%s",
			octopus.config.AccountNumber, octopus.config.ProductCode)
		if octopus.config.Vehicle != "" && octopus.config.ProductCode != "DEU-ELECTRICITY-IO-GO-24" {
			log.INFO.Printf("Vehicle charging optimization only available for DEU-ELECTRICITY-IO-GO-24 tariff")
		}
	}

	done := make(chan error)
	go octopus.run(done)
	err := <-done

	// Register with startup detector for cache-aware vehicle integration
	if octopus.config.Vehicle != "" && octopus.config.ProductCode == "DEU-ELECTRICITY-IO-GO-24" {
		startupDetector.Register(octopus)
	}

	return octopus, err
}

// startVehicleIntegrationIfNeeded starts vehicle integration if not already started
func (o *OctopusGermany) startVehicleIntegrationIfNeeded() {
	// Only start if vehicle integration hasn't been started yet
	if o.vehicleIntegrationStarted {
		return
	}

	// Only proceed if we have vehicle configuration and correct tariff
	if o.config.Vehicle == "" || o.config.ProductCode != "DEU-ELECTRICITY-IO-GO-24" {
		return
	}

	o.log.DEBUG.Printf("Starting vehicle integration via startup detector")

	// Start vehicle integration background processing
	go func() {
		defer func() {
			if r := recover(); r != nil {
				o.log.ERROR.Printf("Vehicle integration panic: %v", r)
			}
		}()

		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				o.processPlannedDispatches()
			}
		}
	}()

	o.vehicleIntegrationStarted = true
	o.log.INFO.Printf("Vehicle integration started for %s", o.config.Vehicle)
}

// startStandaloneVehicleIntegration creates an independent vehicle integration service
// that works even when the tariff instance is not created due to caching
func startStandaloneVehicleIntegration() {
	log := util.NewLogger("octopusgermany")
	log.DEBUG.Printf("Starting standalone vehicle integration service")

	// Create standalone service instance
	service, err := NewStandaloneVehicleService()
	if err != nil {
		log.ERROR.Printf("Failed to create standalone vehicle service: %v", err)
		return
	}

	if service == nil {
		log.DEBUG.Printf("No OctopusGermany vehicle integration needed - no suitable configuration found")
		return
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.INFO.Printf("Standalone vehicle integration service started for vehicle '%s' - monitoring for vehicle plans", service.config.Vehicle)

	// Process immediately on startup
	go service.processVehiclePlans()

	for {
		select {
		case <-ticker.C:
			service.processVehiclePlans()
		}
	}
}

// StandaloneVehicleService implements vehicle integration independently of tariff instances
type StandaloneVehicleService struct {
	log      *util.Logger
	config   Config
	token    string
	tokenExp time.Time
	tokenMu  sync.RWMutex
	*request.Helper
}

// NewStandaloneVehicleService creates a standalone vehicle service from EVCC configuration
func NewStandaloneVehicleService() (*StandaloneVehicleService, error) {
	log := util.NewLogger("octopusgermany")

	// Try to read EVCC configuration from common locations
	configPaths := []string{
		"/workspaces/evcc/evcc.yaml",
		"./evcc.yaml",
		"/etc/evcc.yaml",
		"/home/vscode/.evcc/evcc.yaml",
	}

	var configData map[string]interface{}

	for _, path := range configPaths {
		if data, err := readYAMLConfig(path); err == nil {
			configData = data
			log.DEBUG.Printf("Successfully read EVCC config from %s", path)
			break
		}
	}

	if configData == nil {
		return nil, fmt.Errorf("could not read EVCC configuration from any known location")
	}

	// Extract OctopusGermany tariff configuration with vehicle setting
	tariffs, ok := configData["tariffs"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no tariffs section found in config")
	}

	grid, ok := tariffs["grid"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no grid tariff found in config")
	}

	tariffType, ok := grid["type"].(string)
	if !ok || tariffType != "octopusgermany" {
		return nil, fmt.Errorf("grid tariff is not octopusgermany (found: %s)", tariffType)
	}

	// Extract configuration
	config := Config{}
	if email, ok := grid["email"].(string); ok {
		config.Email = email
	}
	if password, ok := grid["password"].(string); ok {
		config.Password = password
	}
	if accountNumber, ok := grid["accountnumber"].(string); ok {
		config.AccountNumber = accountNumber
	}
	if productCode, ok := grid["productcode"].(string); ok {
		config.ProductCode = productCode
	}
	if vehicle, ok := grid["vehicle"].(string); ok {
		config.Vehicle = vehicle
	}

	// Only create service if vehicle integration is configured
	if config.Vehicle == "" || config.ProductCode != "DEU-ELECTRICITY-IO-GO-24" {
		return nil, nil // No error, just no service needed
	}

	// Validate required fields
	if config.Email == "" || config.Password == "" || config.AccountNumber == "" {
		return nil, fmt.Errorf("missing required OctopusGermany configuration (email, password, or accountnumber)")
	}

	log.DEBUG.Printf("Creating standalone vehicle service for vehicle '%s' with account '%s'", config.Vehicle, config.AccountNumber)

	service := &StandaloneVehicleService{
		log:    log,
		config: config,
		Helper: request.NewHelper(log),
	}

	// Perform initial authentication
	if err := service.login(); err != nil {
		return nil, fmt.Errorf("initial authentication failed: %w", err)
	}

	log.INFO.Printf("Standalone vehicle service authenticated successfully for vehicle '%s'", config.Vehicle)

	return service, nil
}

// readYAMLConfig reads and parses a YAML configuration file
func readYAMLConfig(path string) (map[string]interface{}, error) {
	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	// Since we can't parse YAML easily without additional dependencies,
	// we'll read the configuration from environment variables
	// This is a secure approach for the standalone service

	email := os.Getenv("OCTOPUS_EMAIL")
	password := os.Getenv("OCTOPUS_PASSWORD")
	accountNumber := os.Getenv("OCTOPUS_ACCOUNT_NUMBER")
	productCode := os.Getenv("OCTOPUS_PRODUCT_CODE")
	vehicle := os.Getenv("OCTOPUS_VEHICLE")

	// Default values if not set
	if productCode == "" {
		productCode = "DEU-ELECTRICITY-IO-GO-24"
	}

	// Validate required environment variables
	if email == "" || password == "" || accountNumber == "" {
		return nil, fmt.Errorf("required environment variables not set: OCTOPUS_EMAIL, OCTOPUS_PASSWORD, OCTOPUS_ACCOUNT_NUMBER")
	}

	return map[string]interface{}{
		"tariffs": map[string]interface{}{
			"grid": map[string]interface{}{
				"type":          "octopusgermany",
				"email":         email,
				"password":      password,
				"accountnumber": accountNumber,
				"productcode":   productCode,
				"vehicle":       vehicle,
			},
		},
	}, nil
}

// isTokenValid checks if the current token is valid and not expired
func (s *StandaloneVehicleService) isTokenValid() bool {
	s.tokenMu.RLock()
	defer s.tokenMu.RUnlock()

	if s.token == "" {
		return false
	}

	// Check if token has expired (with margin)
	now := time.Now()
	if !s.tokenExp.IsZero() && now.After(s.tokenExp.Add(-tokenRefreshMargin)) {
		s.log.DEBUG.Printf("Token will expire in %v, needs refresh", s.tokenExp.Sub(now))
		return false
	}

	return true
}

// isTokenValid checks if the current token is valid and not expired for OctopusGermany
func (o *OctopusGermany) isTokenValid() bool {
	o.tokenMu.RLock()
	defer o.tokenMu.RUnlock()

	if o.token == "" {
		return false
	}

	// Check if token has expired (with margin)
	now := time.Now()
	if !o.tokenExp.IsZero() && now.After(o.tokenExp.Add(-tokenRefreshMargin)) {
		o.log.DEBUG.Printf("Token will expire in %v, needs refresh", o.tokenExp.Sub(now))
		return false
	}

	return true
}

// parseTokenExpiry extracts expiry time from JWT token
func parseTokenExpiry(token string) time.Time {
	// Simple JWT parsing without signature verification (like Python implementation)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}
	}

	// Decode payload
	payload := parts[1]
	// Add padding if needed
	for len(payload)%4 != 0 {
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return time.Time{}
	}

	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return time.Time{}
	}

	if claims.Exp == 0 {
		return time.Time{}
	}

	return time.Unix(claims.Exp, 0)
}

// ensureValidToken ensures we have a valid token, refreshing if necessary
func (s *StandaloneVehicleService) ensureValidToken() error {
	if s.isTokenValid() {
		return nil
	}

	s.log.DEBUG.Println("Token invalid or expired, refreshing...")
	return s.login()
}

// ensureValidToken ensures we have a valid token, refreshing if necessary for OctopusGermany
func (o *OctopusGermany) ensureValidToken() error {
	if o.isTokenValid() {
		return nil
	}

	o.log.DEBUG.Println("Token invalid or expired, refreshing...")
	return o.login()
}
func (s *StandaloneVehicleService) login() error {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()

	// Clear existing token when explicitly refreshing
	s.token = ""
	s.tokenExp = time.Time{}

	const loginQuery = `mutation obtainKrakenToken($email: String!, $password: String!) {
		obtainKrakenToken(input: {email: $email, password: $password}) {
			token
		}
	}`

	payload := map[string]interface{}{
		"query": loginQuery,
		"variables": map[string]interface{}{
			"email":    s.config.Email,
			"password": s.config.Password,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal login payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.oeg-kraken.energy/v1/graphql/", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}

	var result struct {
		Data struct {
			ObtainKrakenToken struct {
				Token string `json:"token"`
			} `json:"obtainKrakenToken"`
		} `json:"data"`
		Errors json.RawMessage `json:"errors"`
	}

	if err := json.Unmarshal(b, &result); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	if result.Data.ObtainKrakenToken.Token == "" {
		return fmt.Errorf("no token received from login")
	}

	s.token = result.Data.ObtainKrakenToken.Token
	s.tokenExp = parseTokenExpiry(s.token)

	if s.tokenExp.IsZero() {
		// Fallback: Set expiry to auto-refresh interval from now
		s.tokenExp = time.Now().Add(tokenAutoRefreshTime)
		s.log.WARN.Println("Failed to parse token expiry, using fallback expiry")
	} else {
		s.log.DEBUG.Printf("Token valid until %v", s.tokenExp)
	}

	return nil
}

// authHeader returns the Authorization header value for the standalone service
func (s *StandaloneVehicleService) authHeader() string {
	s.tokenMu.RLock()
	defer s.tokenMu.RUnlock()

	if strings.HasPrefix(s.token, "JWT ") || strings.HasPrefix(s.token, "Bearer ") {
		return s.token
	}
	return "JWT " + s.token
}

// processVehiclePlans processes planned dispatches and sets vehicle charging plans
func (s *StandaloneVehicleService) processVehiclePlans() {
	s.log.DEBUG.Printf("Processing vehicle plans for %s", s.config.Vehicle)

	// Ensure valid token before starting
	if err := s.ensureValidToken(); err != nil {
		s.log.WARN.Printf("Failed to ensure valid token: %v", err)
		return
	}

	// Fetch planned dispatches
	dispatches, err := s.fetchPlannedDispatches()
	if err != nil {
		s.log.WARN.Printf("Failed to fetch planned dispatches: %v", err)
		return
	}

	// Process dispatches same as main implementation
	currentTime := time.Now()
	var nextDispatch *PlannedDispatch
	tolerance := 5 * time.Minute

	for _, dispatch := range dispatches {
		endTime, err := time.Parse(time.RFC3339, dispatch.End)
		if err != nil {
			s.log.WARN.Printf("Failed to parse dispatch end time %s: %v", dispatch.End, err)
			continue
		}

		if endTime.After(currentTime.Add(tolerance)) {
			nextDispatch = &dispatch
			break
		}
	}

	// If no future dispatches found, clear any existing plan
	if nextDispatch == nil {
		s.log.DEBUG.Printf("No future planned dispatches found - clearing any existing vehicle plan")
		if err := s.clearVehicleChargingPlan(); err != nil {
			s.log.WARN.Printf("Failed to clear vehicle charging plan: %v", err)
		}
		return
	}

	// Parse the end time for the charging plan
	endTime, err := time.Parse(time.RFC3339, nextDispatch.End)
	if err != nil {
		s.log.WARN.Printf("Failed to parse dispatch end time: %v", err)
		return
	}

	// Get target SOC from Octopus device data
	targetSoc, err := s.getVehicleTargetSoc()
	if err != nil {
		s.log.WARN.Printf("Failed to get vehicle target SOC from Octopus devices: %v", err)
		return
	}

	s.log.INFO.Printf("Setting vehicle charging plan: end=%s, deltaKwh=%.2f, targetSoc=%d%%",
		endTime.Format("2006-01-02 15:04:05"), nextDispatch.GetDeltaKwh(), targetSoc)

	if err := s.setVehicleChargingPlan(targetSoc, endTime); err != nil {
		s.log.WARN.Printf("Failed to set vehicle charging plan: %v", err)
	}
}

// fetchPlannedDispatches fetches planned dispatches for the standalone service
func (s *StandaloneVehicleService) fetchPlannedDispatches() ([]PlannedDispatch, error) {
	// Ensure we have a valid token before making the request
	if err := s.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	retries := 0
	for retries <= maxTokenRetries {
		dispatches, err := s.doFetchPlannedDispatches()
		if err != nil {
			// Check if it's a JWT expired error
			if strings.Contains(err.Error(), "Signature of the JWT has expired") ||
				strings.Contains(err.Error(), "KT-CT-1124") {
				s.log.WARN.Printf("Token expired during fetchPlannedDispatches (retry %d/%d), refreshing...", retries+1, maxTokenRetries)

				// Clear token and force refresh
				s.tokenMu.Lock()
				s.token = ""
				s.tokenExp = time.Time{}
				s.tokenMu.Unlock()

				if loginErr := s.login(); loginErr != nil {
					return nil, fmt.Errorf("failed to refresh token: %w", loginErr)
				}

				retries++
				if retries <= maxTokenRetries {
					time.Sleep(apiRetryDelay)
					continue
				}
			}
		}
		return dispatches, err
	}

	return nil, fmt.Errorf("max retries exceeded for fetchPlannedDispatches")
}

// doFetchPlannedDispatches performs the actual API call
func (s *StandaloneVehicleService) doFetchPlannedDispatches() ([]PlannedDispatch, error) {
	gql := `query ($accountNumber: String!) {
		plannedDispatches(accountNumber: $accountNumber) {
			start
			end
			startDt
			endDt
			delta
			deltaKwh
			meta {
				location
				source
			}
		}
	}`

	payload := map[string]interface{}{
		"query": gql,
		"variables": map[string]interface{}{
			"accountNumber": s.config.AccountNumber,
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "https://api.oeg-kraken.energy/v1/graphql/", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", s.authHeader())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			PlannedDispatches []PlannedDispatch `json:"plannedDispatches"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", result.Errors[0].Message)
	}

	s.log.DEBUG.Printf("Fetched %d planned dispatch(es)", len(result.Data.PlannedDispatches))
	return result.Data.PlannedDispatches, nil
}

// getVehicleTargetSoc gets target SOC for the standalone service
func (s *StandaloneVehicleService) getVehicleTargetSoc() (int, error) {
	devices, err := s.fetchDevices()
	if err != nil {
		return 0, fmt.Errorf("failed to fetch Octopus devices: %v", err)
	}

	if len(devices) == 0 {
		return 0, fmt.Errorf("no devices found in Octopus API")
	}

	for _, device := range devices {
		for _, schedule := range device.Preferences.Schedules {
			if schedule.Max > 0 && schedule.Max <= 100 {
				socValue := int(schedule.Max)
				s.log.TRACE.Printf("Found device '%s' with schedule max SOC: %d%%", device.Name, socValue)
				return socValue, nil
			}
		}
	}

	return 0, fmt.Errorf("no valid max SOC found in any device schedule")
}

// fetchDevices fetches device information for the standalone service
func (s *StandaloneVehicleService) fetchDevices() ([]Device, error) {
	// Ensure we have a valid token before making the request
	if err := s.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	retries := 0
	for retries <= maxTokenRetries {
		devices, err := s.doFetchDevices()
		if err != nil {
			// Check if it's a JWT expired error
			if strings.Contains(err.Error(), "Signature of the JWT has expired") ||
				strings.Contains(err.Error(), "KT-CT-1124") {
				s.log.WARN.Printf("Token expired during fetchDevices (retry %d/%d), refreshing...", retries+1, maxTokenRetries)

				// Clear token and force refresh
				s.tokenMu.Lock()
				s.token = ""
				s.tokenExp = time.Time{}
				s.tokenMu.Unlock()

				if loginErr := s.login(); loginErr != nil {
					return nil, fmt.Errorf("failed to refresh token: %w", loginErr)
				}

				retries++
				if retries <= maxTokenRetries {
					time.Sleep(apiRetryDelay)
					continue
				}
			}
		}
		return devices, err
	}

	return nil, fmt.Errorf("max retries exceeded for fetchDevices")
}

// doFetchDevices performs the actual API call
func (s *StandaloneVehicleService) doFetchDevices() ([]Device, error) {
	gql := `query ($accountNumber: String!) {
		devices(accountNumber: $accountNumber) {
			deviceType
			name
			preferences {
				schedules {
					dayOfWeek
					max
					min
					time
				}
				targetType
				unit
			}
		}
	}`

	payload := map[string]interface{}{
		"query": gql,
		"variables": map[string]interface{}{
			"accountNumber": s.config.AccountNumber,
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "https://api.oeg-kraken.energy/v1/graphql/", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", s.authHeader())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("devices API request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			Devices []Device `json:"devices"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", result.Errors[0].Message)
	}

	return result.Data.Devices, nil
}

// setVehicleChargingPlan sets charging plan for the standalone service
func (s *StandaloneVehicleService) setVehicleChargingPlan(soc int, timestamp time.Time) error {
	// Ensure token is valid before making EVCC API call
	if err := s.ensureValidToken(); err != nil {
		s.log.WARN.Printf("Token validation failed before setting vehicle plan: %v", err)
		// Continue anyway since this is EVCC API call, not Octopus API
	}

	encodedTimestamp := url.QueryEscape(timestamp.UTC().Format("2006-01-02T15:04:05.000Z"))
	url := fmt.Sprintf("http://127.0.0.1:7070/api/vehicles/%s/plan/soc/%d/%s",
		s.config.Vehicle, soc, encodedTimestamp)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to set vehicle charging plan: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("EVCC API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	s.log.INFO.Printf("Successfully set vehicle %s charging plan: SOC=%d%% at %s",
		s.config.Vehicle, soc, timestamp.Format("2006-01-02 15:04:05"))
	return nil
}

// clearVehicleChargingPlan clears charging plan for the standalone service
func (s *StandaloneVehicleService) clearVehicleChargingPlan() error {
	// Ensure token is valid before making EVCC API call
	if err := s.ensureValidToken(); err != nil {
		s.log.WARN.Printf("Token validation failed before clearing vehicle plan: %v", err)
		// Continue anyway since this is EVCC API call, not Octopus API
	}

	deleteURL := fmt.Sprintf("http://127.0.0.1:7070/api/vehicles/%s/plan/soc", s.config.Vehicle)

	deleteReq, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	deleteResp, err := client.Do(deleteReq)
	if err != nil {
		return fmt.Errorf("failed to delete vehicle plan: %w", err)
	}
	defer deleteResp.Body.Close()

	s.log.INFO.Printf("Successfully cleared vehicle %s charging plan", s.config.Vehicle)
	return nil
}

// login authenticates with Octopus Germany API using GraphQL.
func (o *OctopusGermany) login() error {
	o.tokenMu.Lock()
	defer o.tokenMu.Unlock()

	// Clear existing token when explicitly refreshing
	o.token = ""
	o.tokenExp = time.Time{}

	const loginQuery = `mutation obtainKrakenToken($email: String!, $password: String!) {
		obtainKrakenToken(input: {email: $email, password: $password}) {
			token
		}
	}`

	payload := map[string]interface{}{
		"query": loginQuery,
		"variables": map[string]interface{}{
			"email":    o.config.Email,
			"password": o.config.Password,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal login payload: %w", err)
	}

	o.log.TRACE.Printf("Login GraphQL query: %s", string(body))

	req, err := http.NewRequest("POST", "https://api.oeg-kraken.energy/v1/graphql/", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}

	o.log.TRACE.Printf("Login response: %s", string(b))

	var result struct {
		Data struct {
			ObtainKrakenToken struct {
				Token string `json:"token"`
			} `json:"obtainKrakenToken"`
		} `json:"data"`
		Errors json.RawMessage `json:"errors"`
	}

	if err := json.Unmarshal(b, &result); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	if result.Data.ObtainKrakenToken.Token == "" {
		if len(result.Errors) > 0 {
			o.log.INFO.Printf("Login GraphQL errors: %s", string(result.Errors))
		} else {
			o.log.INFO.Printf("Login response without token: %s", string(b))
		}
		return fmt.Errorf("no token received from login")
	}

	o.token = result.Data.ObtainKrakenToken.Token
	o.tokenExp = parseTokenExpiry(o.token)

	if o.tokenExp.IsZero() {
		// Fallback: Set expiry to auto-refresh interval from now
		o.tokenExp = time.Now().Add(tokenAutoRefreshTime)
		o.log.WARN.Println("Failed to parse token expiry, using fallback expiry")
	} else {
		o.log.DEBUG.Printf("Token valid until %v", o.tokenExp)
	}

	o.log.TRACE.Println("Authenticated successfully, token received.")
	return nil
}

// authHeader returns the Authorization header value, prefixing with JWT if missing
func (o *OctopusGermany) authHeader() string {
	o.tokenMu.RLock()
	defer o.tokenMu.RUnlock()

	if strings.HasPrefix(o.token, "JWT ") || strings.HasPrefix(o.token, "Bearer ") {
		return o.token
	}
	return "JWT " + o.token
}

func (o *OctopusGermany) run(done chan error) {
	var once sync.Once

	// Initial login
	if err := o.login(); err != nil {
		once.Do(func() { done <- err })
		return
	}

	// Close done channel immediately to not block EVCC startup
	once.Do(func() { close(done) })

	// Start timer with first tick after 1 minute
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Vehicle integration will be started from Rates() method to ensure it runs after EVCC restart

	// Fetch initial rates in background after a short delay
	go func() {
		time.Sleep(10 * time.Second) // Brief delay to ensure EVCC is starting up

		var rates api.Rates
		if err := backoff.Retry(func() error {
			var err error
			rates, err = o.fetchRatesGraphQL()
			return backoffPermanentError(err)
		}, bo()); err != nil {
			o.log.ERROR.Printf("Failed to fetch initial rates: %v", err)
		} else {
			mergeRates(o.data, rates)
			o.log.DEBUG.Printf("Initial rates loaded successfully")
		}
	}()

	for {
		select {
		case <-ticker.C:
			var rates api.Rates

			if err := backoff.Retry(func() error {
				var err error
				rates, err = o.fetchRatesGraphQL()
				return backoffPermanentError(err)
			}, bo()); err != nil {
				o.log.ERROR.Println(err)
				continue
			}

			mergeRates(o.data, rates)

			// Process planned dispatches periodically if vehicle integration has started
			if o.vehicleIntegrationStarted {
				// Process vehicle plans immediately when rates are updated
				go o.processPlannedDispatches()
			}
		}
	}
}

// Rates implements the api.Tariff interface
func (o *OctopusGermany) Rates() (api.Rates, error) {
	currentTime := time.Now()

	// Always check if vehicle integration should be started/restarted
	// This ensures it runs even when tariff instance is cached across EVCC restarts
	if o.config.Vehicle != "" && o.config.ProductCode == "DEU-ELECTRICITY-IO-GO-24" && !o.vehicleIntegrationChecked {
		o.vehicleIntegrationChecked = true

		// Start vehicle integration if this instance is fresh (created within last 2 minutes)
		// OR if this is the first call to Rates() after EVCC startup (handles caching proxy case)
		instanceAge := currentTime.Sub(o.startupTime)

		// If this is the first Rates() call, treat it as a potential startup scenario
		isFirstRatesCall := o.lastRatesCall.IsZero()

		if (instanceAge <= 2*time.Minute || isFirstRatesCall) && !o.vehicleIntegrationStarted {
			o.vehicleIntegrationStarted = true
			o.log.DEBUG.Printf("Starting vehicle integration (instance age: %v, first rates call: %v)",
				instanceAge, isFirstRatesCall)

			// Start integration in background with delay
			go func() {
				time.Sleep(15 * time.Second) // Wait for EVCC API to be ready
				o.processPlannedDispatches()
			}()
		}
	}

	// Update last rates call timestamp
	o.lastRatesCall = currentTime

	var res api.Rates
	err := o.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
} // Type implements the api.Tariff interface
func (o *OctopusGermany) Type() api.TariffType {
	return api.TariffTypePriceForecast
}

// DiscoveryResult holds discovered account and product info.
type DiscoveryResult struct {
	AccountNumber string
	ProductCodes  []ProductInfo
}

type ProductInfo struct {
	Code     string
	FullName string
}

// PlannedDispatch represents a planned charging dispatch from Octopus Germany API.
type PlannedDispatch struct {
	Start    string          `json:"start"`
	End      string          `json:"end"`
	StartDt  string          `json:"startDt"`
	EndDt    string          `json:"endDt"`
	Delta    json.RawMessage `json:"delta"`    // API returns as string or number, parse manually
	DeltaKwh json.RawMessage `json:"deltaKwh"` // API returns as string or number, parse manually
	Meta     struct {
		Location string `json:"location"`
		Source   string `json:"source"`
	} `json:"meta"`
}

// GetDelta returns the delta value as float64, parsing from string or number
func (pd *PlannedDispatch) GetDelta() float64 {
	// Try to unmarshal as float64 first
	var numVal float64
	if err := json.Unmarshal(pd.Delta, &numVal); err == nil {
		return numVal
	}

	// Try to unmarshal as string and parse
	var strVal string
	if err := json.Unmarshal(pd.Delta, &strVal); err == nil {
		if val, err := strconv.ParseFloat(strVal, 64); err == nil {
			return val
		}
	}
	return 0
}

// GetDeltaKwh returns the deltaKwh value as float64, parsing from string or number
func (pd *PlannedDispatch) GetDeltaKwh() float64 {
	// Try to unmarshal as float64 first
	var numVal float64
	if err := json.Unmarshal(pd.DeltaKwh, &numVal); err == nil {
		return numVal
	}

	// Try to unmarshal as string and parse
	var strVal string
	if err := json.Unmarshal(pd.DeltaKwh, &strVal); err == nil {
		if val, err := strconv.ParseFloat(strVal, 64); err == nil {
			return val
		}
	}
	return 0
}

// discovery fetches and returns all available accounts and product codes for the user.
func (o *OctopusGermany) discovery() ([]DiscoveryResult, error) {
	gql := `query { viewer { accounts { number ... on AccountType { allProperties { electricityMalos { agreements { product { code fullName } } } } } } } }`
	payload := map[string]interface{}{
		"query": gql,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "https://api.oeg-kraken.energy/v1/graphql/", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", o.authHeader())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("discovery request failed: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read discovery response: %w", err)
	}

	o.log.TRACE.Printf("Discovery response: %s", string(b))

	var result struct {
		Data struct {
			Viewer struct {
				Accounts []struct {
					Number        string `json:"number"`
					AllProperties []struct {
						ElectricityMalos []struct {
							Agreements []struct {
								Product struct {
									Code     string `json:"code"`
									FullName string `json:"fullName"`
								} `json:"product"`
							} `json:"agreements"`
						} `json:"electricityMalos"`
					} `json:"allProperties"`
				} `json:"accounts"`
			} `json:"viewer"`
		} `json:"data"`
	}

	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("failed to parse discovery response: %w", err)
	}

	var results []DiscoveryResult
	for _, account := range result.Data.Viewer.Accounts {
		dr := DiscoveryResult{
			AccountNumber: account.Number,
		}

		productSet := make(map[string]ProductInfo)
		for _, prop := range account.AllProperties {
			for _, malo := range prop.ElectricityMalos {
				for _, agreement := range malo.Agreements {
					if agreement.Product.Code != "" {
						productSet[agreement.Product.Code] = ProductInfo{
							Code:     agreement.Product.Code,
							FullName: agreement.Product.FullName,
						}
					}
				}
			}
		}

		for _, product := range productSet {
			dr.ProductCodes = append(dr.ProductCodes, product)
		}

		if len(dr.ProductCodes) > 0 {
			results = append(results, dr)
		}
	}

	return results, nil
}

// discoverAccountProducts fetches available products for a specific account
func (o *OctopusGermany) discoverAccountProducts(accountNumber string) ([]ProductInfo, error) {
	gql := `query($accountNumber: String!) {
		account(accountNumber: $accountNumber) {
			allProperties {
				electricityMalos {
					agreements {
						product {
							code
							fullName
						}
						validFrom
						validTo
					}
				}
			}
		}
	}`

	payload := map[string]interface{}{
		"query": gql,
		"variables": map[string]interface{}{
			"accountNumber": accountNumber,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	o.log.TRACE.Printf("Account products GraphQL query: %s", string(body))

	req, err := http.NewRequest("POST", "https://api.oeg-kraken.energy/v1/graphql/", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", o.authHeader())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	o.log.TRACE.Printf("Account products response: %s", string(b))

	var result struct {
		Data struct {
			Account struct {
				AllProperties []struct {
					ElectricityMalos []struct {
						Agreements []struct {
							Product struct {
								Code     string `json:"code"`
								FullName string `json:"fullName"`
							} `json:"product"`
							ValidFrom string `json:"validFrom"`
							ValidTo   string `json:"validTo"`
						} `json:"agreements"`
					} `json:"electricityMalos"`
				} `json:"allProperties"`
			} `json:"account"`
		} `json:"data"`
		Errors json.RawMessage `json:"errors"`
	}

	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Errors) > 0 {
		o.log.INFO.Printf("GraphQL errors: %s", string(result.Errors))
		return nil, fmt.Errorf("GraphQL errors: %s", string(result.Errors))
	}

	productSet := make(map[string]ProductInfo)
	activeProducts := make(map[string]ProductInfo)
	now := time.Now()

	// Process agreements and categorize active vs all products
	for _, prop := range result.Data.Account.AllProperties {
		for _, malo := range prop.ElectricityMalos {
			for _, agreement := range malo.Agreements {
				product := ProductInfo{
					Code:     agreement.Product.Code,
					FullName: agreement.Product.FullName,
				}

				if product.Code != "" {
					productSet[product.Code] = product

					// Check if this agreement is currently active
					if agreement.ValidFrom != "" {
						if validFrom, err := time.Parse(time.RFC3339, agreement.ValidFrom); err == nil {
							isActive := now.After(validFrom)

							// Check ValidTo if present
							if agreement.ValidTo != "" {
								if validTo, err := time.Parse(time.RFC3339, agreement.ValidTo); err == nil {
									isActive = isActive && now.Before(validTo)
								}
							}

							if isActive {
								activeProducts[product.Code] = product
								o.log.TRACE.Printf("Found active product: %s (%s)", product.Code, product.FullName)
							}
						}
					} else {
						// No ValidFrom means it's likely active
						activeProducts[product.Code] = product
						o.log.TRACE.Printf("Found product without ValidFrom (likely active): %s (%s)", product.Code, product.FullName)
					}
				}
			}
		}
	}

	// Return active products first, then all products as fallback
	var products []ProductInfo
	if len(activeProducts) > 0 {
		for _, product := range activeProducts {
			products = append(products, product)
		}
		o.log.TRACE.Printf("Returning %d active products", len(products))
	} else {
		for _, product := range productSet {
			products = append(products, product)
		}
		o.log.TRACE.Printf("No active products found, returning all %d products", len(products))
	}

	return products, nil
}

// parseCents robustly parses a numeric value to float64, handling multiple formats.
func parseCents(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
	case json.Number:
		if f, err := val.Float64(); err == nil {
			return f, true
		}
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	}
	return 0, false
}

// fetchRatesGraphQL fetches rates from the Octopus Germany GraphQL API.
func (o *OctopusGermany) fetchRatesGraphQL() (api.Rates, error) {
	// Use retry logic for JWT token refresh
	var retries int
	for {
		rates, err := o.doFetchRatesGraphQL()
		if err != nil {
			// Check for JWT expiry errors and retry with token refresh
			if retries < maxTokenRetries &&
				(strings.Contains(err.Error(), "Signature of the JWT has expired") ||
					strings.Contains(err.Error(), "KT-CT-1124")) {
				o.log.WARN.Printf("Token expired during fetchRatesGraphQL (retry %d/%d), refreshing...", retries+1, maxTokenRetries)

				// Clear token and force refresh
				o.tokenMu.Lock()
				o.token = ""
				o.tokenExp = time.Time{}
				o.tokenMu.Unlock()

				if loginErr := o.login(); loginErr != nil {
					return nil, fmt.Errorf("failed to refresh token: %w", loginErr)
				}

				retries++
				time.Sleep(apiRetryDelay)
				continue
			}
		}
		return rates, err
	}
}

func (o *OctopusGermany) doFetchRatesGraphQL() (api.Rates, error) {
	// Validate configuration
	if o.config.AccountNumber == "" {
		return nil, fmt.Errorf("accountnumber is required for fetching rates")
	}

	gql := `query($accountNumber: String!) {
		account(accountNumber: $accountNumber) {
			allProperties {
				electricityMalos {
					agreements {
						product {
							code
						}
						unitRateInformation {
							... on TimeOfUseProductUnitRateInformation {
								rates {
									latestGrossUnitRateCentsPerKwh
									timeslotActivationRules {
										activeFromTime
										activeToTime
									}
									timeslotName
								}
							}
						}
					}
				}
			}
		}
	}`

	payload := map[string]interface{}{
		"query": gql,
		"variables": map[string]interface{}{
			"accountNumber": o.config.AccountNumber,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rates payload: %w", err)
	}

	o.log.TRACE.Printf("Rates GraphQL query: %s", string(body))

	req, err := http.NewRequest("POST", "https://api.oeg-kraken.energy/v1/graphql/", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create rates request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", o.authHeader())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rates request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rates request failed with status %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read rates response: %w", err)
	}

	o.log.TRACE.Printf("Rates response: %s", string(b))

	// Parse response
	var result struct {
		Data struct {
			Account struct {
				AllProperties []struct {
					ElectricityMalos []struct {
						Agreements []struct {
							Product struct {
								Code string `json:"code"`
							} `json:"product"`
							UnitRateInformation struct {
								Rates []struct {
									LatestGrossUnitRateCentsPerKwh interface{} `json:"latestGrossUnitRateCentsPerKwh"`
									TimeslotActivationRules        []struct {
										ActiveFromTime string `json:"activeFromTime"`
										ActiveToTime   string `json:"activeToTime"`
									} `json:"timeslotActivationRules"`
									TimeslotName string `json:"timeslotName"`
								} `json:"rates"`
							} `json:"unitRateInformation"`
						} `json:"agreements"`
					} `json:"electricityMalos"`
				} `json:"allProperties"`
			} `json:"account"`
		} `json:"data"`
		Errors json.RawMessage `json:"errors"`
	}

	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("failed to parse rates response: %w", err)
	}

	if len(result.Errors) > 0 {
		o.log.INFO.Printf("Rates GraphQL errors: %s", string(result.Errors))
		return nil, fmt.Errorf("GraphQL errors: %s", string(result.Errors))
	}

	// Find the matching agreement
	var matchingRates []struct {
		LatestGrossUnitRateCentsPerKwh interface{} `json:"latestGrossUnitRateCentsPerKwh"`
		TimeslotActivationRules        []struct {
			ActiveFromTime string `json:"activeFromTime"`
			ActiveToTime   string `json:"activeToTime"`
		} `json:"timeslotActivationRules"`
		TimeslotName string `json:"timeslotName"`
	}

	for _, prop := range result.Data.Account.AllProperties {
		for _, malo := range prop.ElectricityMalos {
			for _, agreement := range malo.Agreements {
				if agreement.Product.Code == o.config.ProductCode {
					matchingRates = agreement.UnitRateInformation.Rates
					break
				}
			}
			if len(matchingRates) > 0 {
				break
			}
		}
		if len(matchingRates) > 0 {
			break
		}
	}

	if len(matchingRates) == 0 {
		return nil, fmt.Errorf("no TimeOfUse rates found for product code %s", o.config.ProductCode)
	}

	o.log.TRACE.Printf("Found %d rate periods for product %s", len(matchingRates), o.config.ProductCode)

	// Build rate schedule based on timeslot activation rules
	type TimeSlot struct {
		Rate         float64
		ActiveFrom   string // "HH:MM" format
		ActiveTo     string // "HH:MM" format
		TimeslotName string
	}

	var timeSlots []TimeSlot

	// Extract rates and their activation times
	for _, rate := range matchingRates {
		if rateVal, ok := parseCents(rate.LatestGrossUnitRateCentsPerKwh); ok {
			rateValue := rateVal / 100.0

			o.log.TRACE.Printf("Found rate %.4f EUR/kWh for timeslot '%s'", rateValue, rate.TimeslotName)

			// Get activation rules for this rate
			for _, rule := range rate.TimeslotActivationRules {
				timeSlots = append(timeSlots, TimeSlot{
					Rate:         rateValue,
					ActiveFrom:   rule.ActiveFromTime,
					ActiveTo:     rule.ActiveToTime,
					TimeslotName: rate.TimeslotName,
				})
				o.log.TRACE.Printf("  Active from %s to %s", rule.ActiveFromTime, rule.ActiveToTime)
			}
		}
	}

	if len(timeSlots) == 0 {
		return nil, fmt.Errorf("no valid time slots found in rate data")
	}

	// Helper function to parse "HH:MM" or "HH:MM:SS" time format
	parseTimeSlot := func(timeStr string) (int, int, error) {
		parts := strings.Split(timeStr, ":")
		if len(parts) < 2 || len(parts) > 3 {
			return 0, 0, fmt.Errorf("invalid time format: %s", timeStr)
		}
		hour, err1 := strconv.Atoi(parts[0])
		minute, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return 0, 0, fmt.Errorf("invalid time format: %s", timeStr)
		}
		return hour, minute, nil
	}

	// Helper function to find rate for a specific hour
	getRateForHour := func(hour int) (float64, string, error) {
		for _, slot := range timeSlots {
			fromHour, _, err1 := parseTimeSlot(slot.ActiveFrom)
			toHour, _, err2 := parseTimeSlot(slot.ActiveTo)
			if err1 != nil || err2 != nil {
				o.log.TRACE.Printf("Failed to parse time slot %s-%s: %v, %v", slot.ActiveFrom, slot.ActiveTo, err1, err2)
				continue
			}

			// Special handling for overnight periods where toTime is "00:00:00"
			// This means the period goes from fromTime until midnight the next day
			if toHour == 0 && fromHour != 0 {
				// Period runs from fromHour to 24:00 (midnight)
				if hour >= fromHour {
					o.log.TRACE.Printf("Hour %d matches overnight slot %s (%s-%s)", hour, slot.TimeslotName, slot.ActiveFrom, slot.ActiveTo)
					return slot.Rate, slot.TimeslotName, nil
				}
			} else {
				// Normal time period (e.g., 00:00 to 05:00)
				if hour >= fromHour && hour < toHour {
					o.log.TRACE.Printf("Hour %d matches slot %s (%s-%s)", hour, slot.TimeslotName, slot.ActiveFrom, slot.ActiveTo)
					return slot.Rate, slot.TimeslotName, nil
				}
			}
		}
		return 0, "", fmt.Errorf("no rate found for hour %d", hour)
	}

	// Build 72 hours of rates starting from now
	now := time.Now()

	// Load German timezone explicitly (handles both CET/CEST automatically)
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		o.log.TRACE.Printf("Failed to load Europe/Berlin timezone, using local: %v", err)
		loc = now.Location()
	}

	// Get current time in German timezone
	nowInGermany := now.In(loc)
	startOfDay := time.Date(nowInGermany.Year(), nowInGermany.Month(), nowInGermany.Day(), 0, 0, 0, 0, loc)

	var rates api.Rates
	for hour := 0; hour < 72; hour++ { // 72 hours forecast
		start := startOfDay.Add(time.Duration(hour) * time.Hour)
		end := start.Add(time.Hour)

		hourOfDay := start.Hour()
		rateValue, timeslotName, err := getRateForHour(hourOfDay)
		if err != nil {
			o.log.TRACE.Printf("Warning: %v, skipping hour %d", err, hourOfDay)
			continue
		}

		rates = append(rates, api.Rate{
			Start: start, // Already in local time
			End:   end,   // Already in local time
			Value: rateValue,
		})

		// Only log first few hours to avoid log spam
		if hour < 5 {
			o.log.TRACE.Printf("Hour %02d:00 in %s period = %.4f EUR/kWh", hourOfDay, timeslotName, rateValue)
		}
	} // Sort by start time
	sort.Slice(rates, func(i, j int) bool {
		return rates[i].Start.Before(rates[j].Start)
	})

	o.log.DEBUG.Printf("Built %d rate entries", len(rates))

	return rates, nil
}

// fetchPlannedDispatches fetches planned charging dispatches from Octopus Germany API.
func (o *OctopusGermany) fetchPlannedDispatches() ([]PlannedDispatch, error) {
	if o.config.AccountNumber == "" {
		return nil, fmt.Errorf("account number not configured")
	}

	gql := `query ($accountNumber: String!) {
		plannedDispatches(accountNumber: $accountNumber) {
			start
			end
			startDt
			endDt
			delta
			deltaKwh
			meta {
				location
				source
			}
		}
	}`

	payload := map[string]interface{}{
		"query": gql,
		"variables": map[string]interface{}{
			"accountNumber": o.config.AccountNumber,
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "https://api.oeg-kraken.energy/v1/graphql/", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", o.authHeader())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			PlannedDispatches []PlannedDispatch `json:"plannedDispatches"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	o.log.TRACE.Printf("PlannedDispatches GraphQL raw response: %s", string(respBody))

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", result.Errors[0].Message)
	}

	o.log.DEBUG.Printf("Fetched %d planned dispatch(es)", len(result.Data.PlannedDispatches))
	for i, dispatch := range result.Data.PlannedDispatches {
		o.log.DEBUG.Printf("Dispatch %d: %s to %s, %.2f kWh", i, dispatch.Start, dispatch.End, dispatch.GetDeltaKwh())
	}

	return result.Data.PlannedDispatches, nil
}

// setVehicleChargingPlan sets a charging plan for the configured vehicle via EVCC API.
// Includes retry logic to handle EVCC startup delays.
func (o *OctopusGermany) setVehicleChargingPlan(soc int, timestamp time.Time) error {
	if o.config.Vehicle == "" {
		return fmt.Errorf("no vehicle configured for charging plan integration")
	}

	// Set the new plan directly (no need to delete first)
	// EVCC API endpoint with URL-encoded timestamp
	// Format: POST /api/vehicles/{name}/plan/soc/{soc}/{timestamp}
	// EVCC expects timestamp with milliseconds: 2025-08-10T12:30:00.000Z
	encodedTimestamp := url.QueryEscape(timestamp.UTC().Format("2006-01-02T15:04:05.000Z"))
	url := fmt.Sprintf("http://127.0.0.1:7070/api/vehicles/%s/plan/soc/%d/%s",
		o.config.Vehicle, soc, encodedTimestamp)

	o.log.TRACE.Printf("Setting vehicle plan via POST %s", url)

	// Retry logic for EVCC startup delays
	maxRetries := 5
	retryDelay := time.Second * 10 // Start with 10 seconds

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("POST", url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)

		// Check for connection refused error (EVCC not ready yet)
		if err != nil {
			if attempt < maxRetries && (strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "connect: connection refused")) {
				o.log.WARN.Printf("EVCC API not ready (attempt %d/%d), waiting %v before retry: %v",
					attempt, maxRetries, retryDelay, err)
				time.Sleep(retryDelay)
				retryDelay *= 2 // Exponential backoff
				continue
			}
			return fmt.Errorf("failed to set vehicle charging plan after %d attempts: %w", attempt, err)
		}

		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			if attempt < maxRetries && resp.StatusCode >= 500 {
				o.log.WARN.Printf("EVCC API server error (attempt %d/%d), waiting %v before retry: status %d",
					attempt, maxRetries, retryDelay, resp.StatusCode)
				time.Sleep(retryDelay)
				retryDelay *= 2
				continue
			}
			return fmt.Errorf("EVCC API request failed with status %d: %s", resp.StatusCode, string(respBody))
		}

		o.log.INFO.Printf("Successfully set vehicle %s charging plan: SOC=%d%% at %s (attempt %d, response: %s)",
			o.config.Vehicle, soc, timestamp.Format("2006-01-02 15:04:05"), attempt, string(respBody))

		return nil
	}

	return fmt.Errorf("failed to set vehicle charging plan after %d attempts - EVCC API not available", maxRetries)
}

// clearVehicleChargingPlan clears the charging plan for the configured vehicle via EVCC API.
// Includes retry logic to handle EVCC startup delays.
func (o *OctopusGermany) clearVehicleChargingPlan() error {
	if o.config.Vehicle == "" {
		return fmt.Errorf("no vehicle configured for charging plan integration")
	}

	// Delete existing plan
	deleteURL := fmt.Sprintf("http://127.0.0.1:7070/api/vehicles/%s/plan/soc", o.config.Vehicle)

	o.log.TRACE.Printf("Clearing vehicle plan via DELETE %s", deleteURL)

	// Retry logic for EVCC startup delays
	maxRetries := 3
	retryDelay := time.Second * 5 // Shorter delay for delete operations

	for attempt := 1; attempt <= maxRetries; attempt++ {
		deleteReq, err := http.NewRequest("DELETE", deleteURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create delete request: %w", err)
		}

		client := &http.Client{Timeout: 10 * time.Second}
		deleteResp, err := client.Do(deleteReq)

		// Check for connection refused error (EVCC not ready yet)
		if err != nil {
			if attempt < maxRetries && (strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "connect: connection refused")) {
				o.log.WARN.Printf("EVCC API not ready for plan clearing (attempt %d/%d), waiting %v before retry: %v",
					attempt, maxRetries, retryDelay, err)
				time.Sleep(retryDelay)
				retryDelay *= 2
				continue
			}
			return fmt.Errorf("failed to delete vehicle plan after %d attempts: %w", attempt, err)
		}

		defer deleteResp.Body.Close()

		if deleteResp.StatusCode == http.StatusOK {
			o.log.INFO.Printf("Successfully cleared vehicle %s charging plan (attempt %d)", o.config.Vehicle, attempt)
		} else {
			o.log.WARN.Printf("Delete request returned status %d (plan may not have existed, attempt %d)", deleteResp.StatusCode, attempt)
		}

		return nil
	}

	return fmt.Errorf("failed to clear vehicle charging plan after %d attempts - EVCC API not available", maxRetries)
}

// processPlannedDispatches processes planned dispatches and sets vehicle charging plans if configured.
func (o *OctopusGermany) processPlannedDispatches() {
	o.log.TRACE.Printf("Checking planned dispatches - Product: %s, Vehicle: %s",
		o.config.ProductCode, o.config.Vehicle)

	// Only process for DEU-ELECTRICITY-IO-GO-24 tariff
	if o.config.ProductCode != "DEU-ELECTRICITY-IO-GO-24" {
		o.log.TRACE.Printf("Skipping planned dispatches - wrong product code (need DEU-ELECTRICITY-IO-GO-24, have %s)",
			o.config.ProductCode)
		return
	}

	// Only process if vehicle is configured
	if o.config.Vehicle == "" {
		o.log.TRACE.Printf("Skipping planned dispatches - no vehicle configured")
		return
	}

	currentTime := time.Now()

	o.log.DEBUG.Printf("Processing planned dispatches for vehicle %s", o.config.Vehicle)

	dispatches, err := o.fetchPlannedDispatches()
	if err != nil {
		o.log.WARN.Printf("Failed to fetch planned dispatches: %v", err)
		return
	}

	// Check for future dispatches
	var nextDispatch *PlannedDispatch
	tolerance := 5 * time.Minute

	for _, dispatch := range dispatches {
		endTime, err := time.Parse(time.RFC3339, dispatch.End)
		if err != nil {
			o.log.WARN.Printf("Failed to parse dispatch end time %s: %v", dispatch.End, err)
			continue
		}

		// Consider dispatches that end more than 5 minutes in the future
		if endTime.After(currentTime.Add(tolerance)) {
			nextDispatch = &dispatch
			o.log.TRACE.Printf("Found future dispatch ending at %s (more than 5min from now)", endTime.Format("2006-01-02 15:04:05"))
			break
		} else {
			o.log.TRACE.Printf("Skipping dispatch ending at %s (too close to current time %s)",
				endTime.Format("2006-01-02 15:04:05"), currentTime.Format("2006-01-02 15:04:05"))
		}
	}

	// If no future dispatches found, clear any existing plan immediately
	if nextDispatch == nil {
		o.log.DEBUG.Printf("No future planned dispatches found - clearing any existing vehicle plan")
		if err := o.clearVehicleChargingPlan(); err != nil {
			o.log.WARN.Printf("Failed to clear vehicle charging plan: %v", err)
		} else {
			// Record successful vehicle plan update (clear)
			o.lastVehiclePlanUpdate = time.Now()
		}
		return
	}

	// Parse the end time for the charging plan
	endTime, err := time.Parse(time.RFC3339, nextDispatch.End)
	if err != nil {
		o.log.WARN.Printf("Failed to parse dispatch end time: %v", err)
		return
	}

	// Get target SOC from Octopus device data - no fallback, only set plan if data is available
	targetSoc, err := o.getVehicleTargetSoc()
	if err != nil {
		o.log.WARN.Printf("Failed to get vehicle target SOC from Octopus devices, skipping plan setting: %v", err)
		return
	}

	o.log.INFO.Printf("Setting vehicle charging plan based on planned dispatch: end=%s, deltaKwh=%.2f, targetSoc=%d%%",
		endTime.Format("2006-01-02 15:04:05"), nextDispatch.GetDeltaKwh(), targetSoc)

	if err := o.setVehicleChargingPlan(targetSoc, endTime); err != nil {
		o.log.WARN.Printf("Failed to set vehicle charging plan: %v", err)
	} else {
		// Record successful vehicle plan update
		o.lastVehiclePlanUpdate = time.Now()
	}
}

// getVehicleTargetSoc retrieves the target SOC from Octopus Germany Device API
func (o *OctopusGermany) getVehicleTargetSoc() (int, error) {
	if o.config.Vehicle == "" {
		return 0, fmt.Errorf("no vehicle configured")
	}

	// Get device data from Octopus API to find target SOC
	devices, err := o.fetchDevices()
	if err != nil {
		return 0, fmt.Errorf("failed to fetch Octopus devices: %v", err)
	}

	if len(devices) == 0 {
		return 0, fmt.Errorf("no devices found in Octopus API")
	}

	// Find the device with valid target SOC in schedules - no fallback
	for _, device := range devices {
		// Look for max value in schedules (this is the target SOC)
		for _, schedule := range device.Preferences.Schedules {
			if schedule.Max > 0 && schedule.Max <= 100 {
				socValue := int(schedule.Max) // Convert float64 to int
				o.log.TRACE.Printf("Found device '%s' (type: %s) with schedule max SOC: %d%% (day: %s, time: %s)",
					device.Name, device.DeviceType, socValue, schedule.DayOfWeek, schedule.Time)
				return socValue, nil
			}
		}
	}

	return 0, fmt.Errorf("no valid max SOC found in any device schedule (need value between 1-100%%)")
} // Device represents a device from Octopus Germany API
type Device struct {
	DeviceType  string `json:"deviceType"`
	Name        string `json:"name"`
	Preferences struct {
		Schedules []struct {
			DayOfWeek string  `json:"dayOfWeek"`
			Max       float64 `json:"max"` // This is the target SOC (API returns as float)
			Min       float64 `json:"min"`
			Time      string  `json:"time"`
		} `json:"schedules"`
		TargetType string `json:"targetType"`
		Unit       string `json:"unit"`
	} `json:"preferences"`
}

// fetchDevices fetches device information from Octopus Germany API
func (o *OctopusGermany) fetchDevices() ([]Device, error) {
	if o.config.AccountNumber == "" {
		return nil, fmt.Errorf("account number not configured")
	}

	gql := `query ($accountNumber: String!) {
		devices(accountNumber: $accountNumber) {
			deviceType
			name
			preferences {
				schedules {
					dayOfWeek
					max
					min
					time
				}
				targetType
				unit
			}
		}
	}`

	payload := map[string]interface{}{
		"query": gql,
		"variables": map[string]interface{}{
			"accountNumber": o.config.AccountNumber,
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "https://api.oeg-kraken.energy/v1/graphql/", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", o.authHeader())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Log the error response body for debugging
		errorBody, _ := io.ReadAll(resp.Body)
		o.log.WARN.Printf("Devices API request failed with status %d, response: %s", resp.StatusCode, string(errorBody))
		return nil, fmt.Errorf("devices API request failed with status %d: %s", resp.StatusCode, string(errorBody))
	}

	var result struct {
		Data struct {
			Devices []Device `json:"devices"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	o.log.TRACE.Printf("Devices GraphQL raw response: %s", string(respBody))

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", result.Errors[0].Message)
	}

	o.log.DEBUG.Printf("Fetched %d device(s) from Octopus API", len(result.Data.Devices))

	return result.Data.Devices, nil
}
