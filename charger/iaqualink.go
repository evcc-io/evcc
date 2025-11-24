package charger

// LICENSE
//
// Copyright (c) 2024 andig
//
// This module is NOT covered by the MIT license. All rights reserved.
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/tekkamanendless/iaqualink"
)

func init() {
	registry.AddCtx("iaqualink", NewIAquaLinkFromConfig)
}

type IAquaLink struct {
	*SgReady
	log              *util.Logger
	client           *iaqualink.Client // Cloud mode client
	helper           *request.Helper   // Local mode HTTP helper
	uri              string            // Local mode: device IP/URL
	deviceID         string            // Cloud mode: device ID
	deviceName       string            // Device name/identifier
	features         []string          // Available device features
	localMode        bool              // true if using local IP, false if using cloud
	readModeDisabled bool              // If true, skip mode reading attempts (API limitations)
	mu               sync.Mutex
}

var _ api.ChargerEx = (*IAquaLink)(nil)

// NewIAquaLinkFromConfig creates an IAquaLink charger from generic config
func NewIAquaLinkFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		embed           `mapstructure:",squash"`
		URI             string // Local mode: IP address or URL of the device
		Email, Password string // Cloud mode: IAquaLink credentials
		Device          string // Device name/identifier (required for cloud mode)
	}{
		embed: embed{
			Icon_:     "heatpump",
			Features_: []api.Feature{api.Heating, api.IntegratedDevice},
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// Either URI (local) or Email/Password (cloud) must be provided
	if cc.URI == "" && (cc.Email == "" || cc.Password == "") {
		return nil, errors.New("must provide either uri (local mode) or email/password (cloud mode)")
	}

	if cc.URI != "" && (cc.Email != "" || cc.Password != "") {
		return nil, errors.New("cannot use both uri (local) and email/password (cloud) - choose one mode")
	}

	// For cloud mode, device name is required
	if cc.URI == "" && cc.Device == "" {
		return nil, errors.New("device name is required for cloud mode")
	}

	return NewIAquaLink(ctx, &cc.embed, cc.URI, cc.Email, cc.Password, cc.Device)
}

// NewIAquaLink creates IAquaLink charger
// Supports both local mode (via URI) and cloud mode (via email/password)
func NewIAquaLink(ctx context.Context, embed *embed, uri, email, password, device string) (api.Charger, error) {
	log := util.NewLogger("iaqualink").Redact(email, password, device)

	c := &IAquaLink{
		SgReady:    nil, // will be set after creating mode functions
		log:        log,
		deviceName: device,
	}

	// Determine mode: local (URI) or cloud (email/password)
	if uri != "" {
		// Local mode: direct IP communication
		c.localMode = true
		c.uri = util.DefaultScheme(strings.TrimRight(uri, "/"), "http")
		c.helper = request.NewHelper(log)

		log.INFO.Printf("IAquaLink using local mode: %s", c.uri)

		// For local mode, try to discover device capabilities
		// Most IAquaLink devices support similar local APIs
		c.features = []string{iaqualink.FeatureModeInfo, iaqualink.FeatureStatus}
	} else {
		// Cloud mode: use IAquaLink API
		c.localMode = false
		client := &iaqualink.Client{
			Client: request.NewClient(log),
		}

		// Login to IAquaLink
		loginOutput, err := client.Login(email, password)
		if err != nil {
			return nil, fmt.Errorf("IAquaLink login failed: %w", err)
		}

		// Store authentication tokens
		client.AuthenticationToken = loginOutput.AuthenticationToken
		if loginOutput.UserPoolOAuth.IDToken != "" {
			client.IDToken = loginOutput.UserPoolOAuth.IDToken
		}
		client.UserID = loginOutput.ID.String()

		// Find device by name
		devices, err := client.ListDevices()
		if err != nil {
			return nil, fmt.Errorf("failed to list IAquaLink devices: %w", err)
		}

		deviceID, serialNumber, matchedBy := c.findDevice(devices, device, log)
		if deviceID == "" {
			return nil, fmt.Errorf("device not found in IAquaLink systems (tried matching by serial number and name)")
		}

		log.INFO.Printf("IAquaLink device matched by %s", matchedBy)

		// Try using serial number if ID doesn't work
		// Some API endpoints might require serial number instead of ID
		deviceIdentifiers := []string{deviceID}
		if serialNumber != "" {
			deviceIdentifiers = append(deviceIdentifiers, serialNumber)
		}

		// Get device features to determine available modes
		// Try different identifiers in case one doesn't work
		var featuresOutput *iaqualink.DeviceFeaturesOutput
		var featuresErr error
		for _, identifier := range deviceIdentifiers {
			log.DEBUG.Printf("Trying to get device features")
			featuresOutput, featuresErr = client.DeviceFeatures(identifier)
			if featuresErr == nil {
				log.DEBUG.Printf("Successfully got device features")
				// Update deviceID to the working identifier
				deviceID = identifier
				break
			}
			log.DEBUG.Printf("Failed to get device features: %v", featuresErr)
		}

		if featuresErr != nil {
			// Log as debug if it's a server error (500) - these are often transient
			// Only warn for authentication errors (401) or other client errors
			if isAPIErrorSuppressible(featuresErr) {
				log.DEBUG.Printf("Device features endpoint returned server error (may be unsupported): %v, using default modes", featuresErr)
				// If features endpoint fails with 500, mode reading will likely also fail
				// Set flag to skip mode reading attempts to reduce log noise
				c.readModeDisabled = true
			} else {
				log.WARN.Printf("Failed to get device features with all identifiers: %v, using default modes", featuresErr)
			}
			featuresOutput = &iaqualink.DeviceFeaturesOutput{Features: []string{}}
		}

		c.client = client
		c.deviceID = deviceID
		c.features = featuresOutput.Features

		log.DEBUG.Printf("IAquaLink device features: %v", featuresOutput.Features)
	}

	// Log available modes
	log.INFO.Printf("IAquaLink device supports modes: Boost(3), Smart(2), Eco/Off(1) (mode: %s)", map[bool]string{true: "local", false: "cloud"}[c.localMode])

	// Create mode setter and getter functions
	setMode := func(mode int64) error {
		return c.setMode(ctx, mode)
	}

	getMode := func() (int64, error) {
		return c.getMode(ctx)
	}

	sgr, err := NewSgReady(ctx, embed, setMode, getMode, nil)
	if err != nil {
		return nil, err
	}

	c.SgReady = sgr

	return decorateIAquaLink(c), nil
}

// setMode sets the device mode based on evcc SGReady mode (1/2/3)
func (c *IAquaLink) setMode(ctx context.Context, mode int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if mode < 1 || mode > 3 {
		return fmt.Errorf("invalid mode %d, expected 1/2/3", mode)
	}

	if c.localMode {
		return c.setModeLocal(ctx, mode)
	}
	return c.setModeCloud(ctx, mode)
}

// setModeCloud sets mode using cloud API
func (c *IAquaLink) setModeCloud(ctx context.Context, mode int64) error {
	// Determine available actions based on device features and mode
	actions := c.getActionsForMode(mode)
	if len(actions) == 0 {
		return fmt.Errorf("device does not support mode %d (available features: %v)", mode, c.features)
	}

	// Use DeviceWebSocket to set mode
	// Try each action until one succeeds (some devices may not support all actions)
	var lastErr error
	for _, action := range actions {
		c.log.DEBUG.Printf("Trying to set mode %d with action '%s'", mode, action)
		result, err := c.client.DeviceWebSocket(c.deviceID, action)
		if err == nil {
			c.log.DEBUG.Printf("Successfully set mode %d with action '%s', result: %v", mode, action, result)
			return nil
		}
		lastErr = err
		c.log.DEBUG.Printf("Action '%s' failed: %v, trying next", action, err)
	}

	// All actions failed
	return fmt.Errorf("failed to set IAquaLink mode %d with any action %v: %w", mode, actions, lastErr)
}

// setModeLocal sets mode using local IP API
// Note: Local API endpoints vary by device model/installation. This implementation
// tries common endpoint patterns, but may need device-specific configuration for some installations.
func (c *IAquaLink) setModeLocal(ctx context.Context, mode int64) error {
	// Map evcc mode to IAquaLink local API commands
	modeCommands := map[int64]string{
		1: "eco",   // Dimm
		2: "smart", // Normal
		3: "boost", // Boost
	}

	command := modeCommands[mode]
	if command == "" {
		return fmt.Errorf("invalid mode %d for local API", mode)
	}

	// Try common local API endpoints (endpoints may vary by device model)
	endpoints := []string{
		fmt.Sprintf("%s/api/v1/mode", c.uri),
		fmt.Sprintf("%s/api/mode", c.uri),
		fmt.Sprintf("%s/mode", c.uri),
		fmt.Sprintf("%s/api/v1/command", c.uri),
	}

	data := map[string]string{"mode": command}
	body, _ := json.Marshal(data)

	var lastErr error
	for _, endpoint := range endpoints {
		c.log.DEBUG.Printf("Trying local endpoint: %s with command: %s", endpoint, command)

		req, err := request.New("POST", endpoint, strings.NewReader(string(body)), request.JSONEncoding)
		if err != nil {
			lastErr = err
			continue
		}

		_, err = c.helper.DoBody(req)
		if err == nil {
			c.log.DEBUG.Printf("Successfully set mode %d via local API: %s", mode, endpoint)
			return nil
		}
		lastErr = err
		c.log.DEBUG.Printf("Local endpoint %s failed: %v", endpoint, err)
	}

	return fmt.Errorf("failed to set mode %d via local API: %w", mode, lastErr)
}

// getActionsForMode returns the IAquaLink actions for the given evcc mode
// based on available device features
func (c *IAquaLink) getActionsForMode(mode int64) []string {
	// Common mode mappings (try these in order of preference)
	modeActions := map[int64][]string{
		1: {"eco", "off"},      // Dimm - try eco first, then off
		2: {"smart", "normal"}, // Normal - try smart first, then normal
		3: {"boost"},           // Boost
	}

	actions := modeActions[mode]
	// The DeviceWebSocket will fail if action is not supported, so return all actions
	return actions
}

// getMode gets the current device mode as evcc SGReady mode (1/2/3)
func (c *IAquaLink) getMode(ctx context.Context) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.localMode {
		return c.getModeLocal(ctx)
	}
	return c.getModeCloud(ctx)
}

// getModeCloud gets mode using cloud API
func (c *IAquaLink) getModeCloud(ctx context.Context) (int64, error) {
	// Check if we have a valid client
	if c.client == nil {
		return 2, fmt.Errorf("IAquaLink client not initialized")
	}

	// If mode reading is disabled (due to API limitations), return default immediately
	if c.readModeDisabled {
		return 2, nil // Default to normal mode
	}

	// Try multiple methods to get device state
	// Method 1: Use DeviceSite to get device information
	// Note: DeviceSite currently only provides timezone info, not mode information
	// This is kept for potential future API expansion
	site, err := c.client.DeviceSite(c.deviceID)
	if err != nil {
		// Suppress 401/500 errors as they're likely API limitations
		if !isAPIErrorSuppressible(err) {
			c.log.DEBUG.Printf("DeviceSite failed: %v", err)
		}
	} else if site != nil {
		// DeviceSite doesn't currently provide mode information
		// This is a placeholder for future API expansion
		_ = site
	}

	// Method 2: Use DeviceExecuteReadCommand to read state
	// Try different commands based on available features
	commandsToTry := []string{"state", "status"}

	// If device has mode_info feature, try that first
	if c.hasFeature(iaqualink.FeatureModeInfo) {
		commandsToTry = append([]string{"mode_info"}, commandsToTry...)
	}

	for _, cmd := range commandsToTry {
		values := url.Values{}
		values.Set(cmd, "1")

		output, err := c.client.DeviceExecuteReadCommand(c.deviceID, values)
		if err != nil {
			// Suppress repeated 401/500 errors as they're likely API limitations
			if !isAPIErrorSuppressible(err) {
				c.log.DEBUG.Printf("DeviceExecuteReadCommand '%s' failed: %v", cmd, err)
			}
			continue
		}
		if output != nil && output.Command.Response != "" {
			if mode := c.parseModeFromResponse(output.Command.Response); mode > 0 {
				c.log.DEBUG.Printf("Got mode %d from command '%s'", mode, cmd)
				return mode, nil
			}
		}
	}

	// Fallback: assume normal mode if we can't determine
	// Don't log this every time as it's expected for devices that don't support mode reading
	c.log.DEBUG.Printf("Could not determine device mode from any method, defaulting to normal")
	return 2, nil
}

// getModeLocal gets mode using local IP API
// Note: Local API endpoints vary by device model/installation. This implementation
// tries common endpoint patterns, but may need device-specific configuration for some installations.
func (c *IAquaLink) getModeLocal(ctx context.Context) (int64, error) {
	// Try common local API endpoints to read device state (endpoints may vary by device model)
	endpoints := []string{
		fmt.Sprintf("%s/api/v1/state", c.uri),
		fmt.Sprintf("%s/api/state", c.uri),
		fmt.Sprintf("%s/state", c.uri),
		fmt.Sprintf("%s/api/v1/status", c.uri),
		fmt.Sprintf("%s/api/status", c.uri),
		fmt.Sprintf("%s/status", c.uri),
	}

	for _, endpoint := range endpoints {
		c.log.DEBUG.Printf("Trying local endpoint: %s", endpoint)

		req, err := request.New("GET", endpoint, nil, request.AcceptJSON)
		if err != nil {
			continue
		}

		respBody, err := c.helper.DoBody(req)
		if err == nil && len(respBody) > 0 {
			if mode := c.parseModeFromResponse(string(respBody)); mode > 0 {
				c.log.DEBUG.Printf("Got mode %d from local endpoint: %s", mode, endpoint)
				return mode, nil
			}
		}
	}

	// Fallback: assume normal mode if we can't determine
	c.log.DEBUG.Printf("Could not determine device mode from local API, defaulting to normal")
	return 2, nil
}

// parseModeFromResponse parses mode from device response string
func (c *IAquaLink) parseModeFromResponse(response string) int64 {
	stateStr := strings.ToLower(response)

	// Check for boost mode (highest priority)
	if strings.Contains(stateStr, "boost") {
		return 3
	}

	// Check for normal/smart mode
	if strings.Contains(stateStr, "smart") || strings.Contains(stateStr, "normal") {
		return 2
	}

	// Check for eco/off mode (dimm)
	if strings.Contains(stateStr, "eco") || strings.Contains(stateStr, "off") {
		return 1
	}

	// Try numeric values (some devices use 0/1/2)
	if strings.Contains(stateStr, "\"0\"") || strings.Contains(stateStr, ":0") {
		return 3 // Boost
	}
	if strings.Contains(stateStr, "\"1\"") || strings.Contains(stateStr, ":1") {
		return 1 // Eco
	}
	if strings.Contains(stateStr, "\"2\"") || strings.Contains(stateStr, ":2") {
		return 2 // Smart
	}

	return 0 // Unknown
}

// hasFeature checks if device has a specific feature
func (c *IAquaLink) hasFeature(feature string) bool {
	return slices.Contains(c.features, feature)
}

// findDevice searches for a device in the list by serial number or name
// Returns deviceID, serialNumber, and matchedBy (how it was matched)
func (c *IAquaLink) findDevice(devices iaqualink.ListDevicesOutput, device string, log *util.Logger) (string, string, string) {
	deviceLower := strings.ToLower(device)

	// Try to match device by serial number first (more reliable)
	for _, dev := range devices {
		if strings.EqualFold(dev.SerialNumber, device) {
			log.DEBUG.Printf("Found device by serial number: ID=%d", dev.ID)
			return strconv.Itoa(dev.ID), dev.SerialNumber, "serial number"
		}
	}

	// If not found by serial number, try matching by name
	for _, dev := range devices {
		if strings.Contains(strings.ToLower(dev.Name), deviceLower) {
			log.DEBUG.Printf("Found device by name: ID=%d", dev.ID)
			return strconv.Itoa(dev.ID), dev.SerialNumber, "name"
		}
	}

	return "", "", ""
}

// isAPIErrorSuppressible checks if an error should be suppressed (logged as debug)
// Returns true for common API limitations like 401/500 errors
func isAPIErrorSuppressible(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "401") || strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "UNAUTHORIZED") || strings.Contains(errStr, "SERVER_ERROR")
}
