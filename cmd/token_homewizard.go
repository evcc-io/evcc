package cmd

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	battery "github.com/evcc-io/evcc/meter/homewizard-battery"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/spf13/cobra"
)

type pairedDevice struct {
	Host  string
	Token string
	Type  battery.DeviceType
}

var homeWizardCmd = &cobra.Command{
	Use:   "homewizard",
	Short: "Pair with HomeWizard devices (P1 meter and batteries)",
	Run:   runHomeWizardToken,
}

func init() {
	tokenCmd.AddCommand(homeWizardCmd)
	homeWizardCmd.Flags().String("host", "", "Device hostname or IP (optional - will auto-discover if not provided)")
	homeWizardCmd.Flags().StringP("name", "n", "evcc", "Product name for pairing")
	homeWizardCmd.Flags().Int("timeout", 10, "Discovery timeout in seconds")
}

func runHomeWizardToken(cmd *cobra.Command, args []string) {
	// Parse log levels to enable debug/trace logging if requested
	parseLogLevels()

	host := cmd.Flag("host").Value.String()
	name := cmd.Flag("name").Value.String()
	timeout, _ := cmd.Flags().GetInt("timeout")

	// Validate name according to HomeWizard API requirements
	namePattern := regexp.MustCompile(`^[a-zA-Z0-9\-_/\\# ]{1,40}$`)
	if !namePattern.MatchString(name) {
		log.FATAL.Fatal("Invalid name: must be 1-40 characters (a-z, A-Z, 0-9, -, _, \\, /, #, spaces)")
	}

	var devices []battery.DiscoveredDevice

	if host != "" {
		// Single device mode
		if regexp.MustCompile(`^https?://`).MatchString(host) {
			log.FATAL.Fatal("Host should not contain http:// or https:// prefix")
		}

		devices = []battery.DiscoveredDevice{
			{Host: host, Type: battery.DeviceTypeUnknown},
		}
	} else {
		// Discovery mode
		fmt.Println("HomeWizard Device Discovery")
		fmt.Println("===========================")
		fmt.Println()
		fmt.Printf("Scanning network (max %ds)...\n", timeout)
		fmt.Println()

		devices = discoverInteractively(timeout)

		if len(devices) == 0 {
			log.FATAL.Fatal("No HomeWizard devices found on network 😞")
		}
	}

	fmt.Println()
	fmt.Println("HomeWizard Device Pairing")
	fmt.Println("=========================")
	fmt.Println()
	fmt.Println("Press the button on ALL devices NOW!")
	fmt.Println()

	// Pair all devices in parallel
	paired := pairDevicesParallel(devices, name)

	// Print configuration
	printHomeWizardMultiConfig(paired)
}

type discoverySpinner struct {
	frames []string
	idx    int
	active bool
}

func newSpinner() *discoverySpinner {
	return &discoverySpinner{
		frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		active: true,
	}
}

func (s *discoverySpinner) tick() {
	if s.active {
		s.idx = (s.idx + 1) % len(s.frames)
		fmt.Printf("\r%s Searching...", s.frames[s.idx])
	}
}

func (s *discoverySpinner) clear() {
	fmt.Print("\r\033[K")
}

func (s *discoverySpinner) stop() {
	s.clear()
	s.active = false
}

func printDiscoveredDevice(count int, device battery.DiscoveredDevice) {
	fmt.Printf("  %d. %s (%s) at %s\n", count, device.Instance, device.Type, device.Host)
}

func confirmDevicesFound() bool {
	fmt.Println()
	fmt.Print("Is this everything? [Y/n]: ")

	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))

	if response == "n" || response == "no" {
		fmt.Println()
		fmt.Println("Discovery aborted.")
		fmt.Println("Please ensure all devices are powered on and on the same network, then try again.")
		return false
	}

	return true
}

func discoverInteractively(timeoutSec int) []battery.DiscoveredDevice {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	deviceChan := make(chan battery.DiscoveredDevice, 10)
	done := make(chan struct{})

	// Start discovery
	go func() {
		battery.DiscoverDevices(ctx, func(device battery.DiscoveredDevice) {
			deviceChan <- device
		})
		close(done)
	}()

	devices := make([]battery.DiscoveredDevice, 0)
	spinner := newSpinner()
	spinnerTicker := time.NewTicker(100 * time.Millisecond)
	defer spinnerTicker.Stop()

	// Stop searching after this period of no new devices
	quietPeriod := 3 * time.Second
	var quietTimer <-chan time.Time

	fmt.Printf("\r%s Searching...", spinner.frames[0])

	for {
		select {
		case device := <-deviceChan:
			// Clear spinner and print device
			spinner.clear()
			devices = append(devices, device)
			printDiscoveredDevice(len(devices), device)

			// Start/reset quiet period timer
			quietTimer = time.After(quietPeriod)

		case <-spinnerTicker.C:
			spinner.tick()

		case <-quietTimer:
			// No new devices found recently, stop searching
			cancel()
			<-done // Wait for discovery goroutine to finish
			spinner.stop()

			// Ask user if satisfied with results
			if len(devices) > 0 && !confirmDevicesFound() {
				log.FATAL.Fatal("Not all devices found")
			}
			return devices

		case <-done:
			// Overall timeout reached
			spinner.stop()

			// Ask user if satisfied with results
			if len(devices) > 0 && !confirmDevicesFound() {
				log.FATAL.Fatal("Not all devices found")
			}
			return devices
		}
	}
}

type deviceStatus struct {
	device  battery.DiscoveredDevice
	status  string
	attempt int
	token   string
	err     error
}

func pairDevicesParallel(devices []battery.DiscoveredDevice, name string) []pairedDevice {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	statuses := make([]*deviceStatus, len(devices))
	var statusMu sync.Mutex
	var wg sync.WaitGroup

	// Initialize and print status for each device
	for i := range devices {
		statuses[i] = &deviceStatus{
			device: devices[i],
			status: "waiting...",
		}
		fmt.Printf("[%d] %s: %s\n", i+1, statuses[i].device.Host, statuses[i].status)
	}

	totalLines := len(statuses)

	// Start pairing goroutine for each device
	for i := range devices {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			status := statuses[idx]
			device := devices[idx]

			token, err := pairDeviceWithContext(ctx, device.Host, name, func(attempt int) {
				statusMu.Lock()
				defer statusMu.Unlock()
				status.attempt = attempt
				status.status = fmt.Sprintf("attempt %d/36...", attempt)
				updateStatusLine(idx, status, totalLines)
			})

			statusMu.Lock()
			defer statusMu.Unlock()

			if err != nil {
				status.err = err
				status.status = fmt.Sprintf("✗ FAILED: %v", err)
			} else {
				status.token = token
				status.status = "✓ SUCCESS"
			}
			updateStatusLine(idx, status, totalLines)
		}(i)
	}

	wg.Wait()
	fmt.Println()

	// Build result - pre-allocate with capacity
	paired := make([]pairedDevice, 0, len(devices))
	failedCount := 0

	for _, status := range statuses {
		if status.token != "" {
			paired = append(paired, pairedDevice{
				Host:  status.device.Host,
				Token: status.token,
				Type:  status.device.Type,
			})
		} else {
			failedCount++
		}
	}

	if failedCount > 0 {
		fmt.Printf("\nWarning: %d device(s) failed to pair\n", failedCount)
	}

	return paired
}

func updateStatusLine(line int, status *deviceStatus, totalLines int) {
	// Move cursor up to the line, clear it, and print new status
	fmt.Printf("\033[%dA\r\033[K[%d] %s: %s\033[%dB\r",
		totalLines-line, line+1, status.device.Host, status.status, totalLines-line)
}

func pairDeviceWithContext(ctx context.Context, host, name string, onAttempt func(int)) (string, error) {
	uri := fmt.Sprintf("https://%s", host)

	helper := request.NewHelper(util.NewLogger("homewizard"))
	helper.Client.Transport = transport.Insecure()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	attempt := 0

	for {
		select {
		case <-ticker.C:
			attempt++
			onAttempt(attempt)

			token, err := requestToken(helper, uri, name)
			if err == nil {
				return token, nil
			}

			if !isButtonPressRequired(err) {
				return "", fmt.Errorf("error: %v", err)
			}

		case <-ctx.Done():
			return "", fmt.Errorf("timeout after 3 minutes")
		}
	}
}

func requestToken(helper *request.Helper, uri, name string) (string, error) {
	endpoint := fmt.Sprintf("%s/api/user", uri)

	reqBody := struct {
		Name string `json:"name"`
	}{
		Name: fmt.Sprintf("local/%s", name),
	}

	var res struct {
		Token string `json:"token"`
	}

	req, err := request.New(http.MethodPost, endpoint, request.MarshalJSON(reqBody), request.JSONEncoding)
	if err != nil {
		return "", err
	}

	req.Header.Set("X-Api-Version", "2")

	err = helper.DoJSON(req, &res)
	return res.Token, err
}

func isButtonPressRequired(err error) bool {
	if reqErr, ok := err.(*request.StatusError); ok {
		return reqErr.StatusCode() == http.StatusForbidden
	}
	return false
}

func printHomeWizardMultiConfig(devices []pairedDevice) {
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("Configuration Complete!")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("Add this to your evcc.yaml configuration:")
	fmt.Println()

	// Find P1 meter
	var p1Meter *pairedDevice
	batteries := make([]pairedDevice, 0)

	for i := range devices {
		if devices[i].Type == battery.DeviceTypeP1Meter {
			p1Meter = &devices[i]
		} else if devices[i].Type == battery.DeviceTypeBattery {
			batteries = append(batteries, devices[i])
		}
	}

	if p1Meter == nil && len(devices) > 0 {
		// If we don't know the type, assume the first is P1 meter
		p1Meter = &devices[0]
		batteries = devices[1:]
	}

	fmt.Println("meters:")
	fmt.Println("- name: battery")
	fmt.Println("  type: homewizard-battery")

	if p1Meter != nil {
		fmt.Printf("  host: %s  # P1 meter\n", p1Meter.Host)
		fmt.Printf("  token: %s\n", p1Meter.Token)
	}

	if len(batteries) > 0 {
		fmt.Println("  batteries:")
		for _, bat := range batteries {
			fmt.Printf("  - host: %s\n", bat.Host)
			fmt.Printf("    token: %s\n", bat.Token)
		}
	}

	fmt.Println()
}
