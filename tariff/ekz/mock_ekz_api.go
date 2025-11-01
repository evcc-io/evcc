package ekz

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// MockServer represents the mock EKZ API server
type MockServer struct {
	port int
}

// NewMockServer creates a new mock EKZ API server
func NewMockServer(port int) *MockServer {
	if port == 0 {
		port = 33927
	}
	return &MockServer{port: port}
}

// Start starts the mock server
func (s *MockServer) Start() error {
	http.HandleFunc("/v1/tariffs", s.handleTariffs)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Starting mock EKZ API server on port %d", s.port)
	return http.ListenAndServe(addr, nil)
}

// handleTariffs handles the /v1/tariffs endpoint
func (s *MockServer) handleTariffs(w http.ResponseWriter, r *http.Request) {
	tariffName := r.URL.Query().Get("tariff_name")
	if tariffName == "" {
		http.Error(w, "tariff_name parameter is required", http.StatusBadRequest)
		return
	}

	// Parse start and end parameters if provided
	var startTime, endTime time.Time

	if startStr := r.URL.Query().Get("start"); startStr != "" {
		if startUnix, err := strconv.ParseInt(startStr, 10, 64); err == nil {
			startTime = time.Unix(startUnix, 0)
		}
	}

	if endStr := r.URL.Query().Get("end"); endStr != "" {
		if endUnix, err := strconv.ParseInt(endStr, 10, 64); err == nil {
			endTime = time.Unix(endUnix, 0)
		}
	}

	// Default to current day if no start time provided
	if startTime.IsZero() {
		now := time.Now()
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}

	// Default to current day + 1 future day if no end time provided
	if endTime.IsZero() {
		endTime = startTime.Add(48 * time.Hour) // 2 days of data
	}

	response := s.generateTariffData(tariffName, startTime, endTime)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}


// generateTariffData generates mock tariff data for the specified time range
func (s *MockServer) generateTariffData(tariffName string, start, end time.Time) TariffResponse {
	var prices []PriceEntry

	// Check if tariff is supported
	switch tariffName {
	case "integrated_400ST", "integrated_400F":
		prices = s.generateStaticTariffData(start, end)
	case "integrated_400D":
		prices = s.generateDynamicTariffData(start, end)
	default:
		log.Printf("Unsupported tariff requested: %s", tariffName)
		return TariffResponse{Prices: []PriceEntry{}}
	}

	return TariffResponse{Prices: prices}
}

// generateStaticTariffData generates constant pricing data for static tariffs
func (s *MockServer) generateStaticTariffData(start, end time.Time) []PriceEntry {
	var prices []PriceEntry

	// Static pricing constants (CHF/kWh)
	const (
		electricityRate = 0.20
		gridRate        = 0.08
		regionalRate    = 0.03
		meteringRate    = 0.02
		integratedRate  = electricityRate + gridRate + regionalRate + meteringRate // 0.33
		monthlyFixed    = 5.0                                                      // CHF/M
	)

	// Generate 15-minute intervals
	current := start
	for current.Before(end) {
		intervalEnd := current.Add(15 * time.Minute)

		entry := PriceEntry{
			StartTimestamp: current,
			EndTimestamp:   intervalEnd,
			Electricity: []Rate{
				{Unit: "CHF/kWh", Value: electricityRate},
				{Unit: "CHF/M", Value: monthlyFixed},
			},
			Grid: []Rate{
				{Unit: "CHF/kWh", Value: gridRate},
			},
			RegionalFees: []Rate{
				{Unit: "CHF/kWh", Value: regionalRate},
			},
			Metering: []Rate{
				{Unit: "CHF/kWh", Value: meteringRate},
			},
			Integrated: []Rate{
				{Unit: "CHF/kWh", Value: integratedRate},
				{Unit: "CHF/M", Value: monthlyFixed},
			},
		}

		prices = append(prices, entry)
		current = intervalEnd
	}

	return prices
}

// generateDynamicTariffData generates dynamic pricing data with hourly variations
func (s *MockServer) generateDynamicTariffData(start, end time.Time) []PriceEntry {
	var prices []PriceEntry

	// Base rates (CHF/kWh) - these are constant
	const (
		baseGridRate     = 0.08
		baseRegionalRate = 0.03
		baseMeteringRate = 0.02
		monthlyFixed     = 5.0 // CHF/M
	)

	// Dynamic electricity rates by hour (24 hours) - simulating daily pattern
	electricityRates := []float64{
		0.15, 0.14, 0.13, 0.13, 0.14, 0.16, // 00-05: low night rates
		0.22, 0.25, 0.24, 0.20, 0.18, 0.17, // 06-11: morning peak
		0.18, 0.19, 0.18, 0.17, 0.18, 0.19, // 12-17: afternoon
		0.23, 0.26, 0.24, 0.21, 0.18, 0.16, // 18-23: evening peak
	}

	// Generate 15-minute intervals
	current := start
	for current.Before(end) {
		intervalEnd := current.Add(15 * time.Minute)

		// Get hourly rate based on current hour
		hour := current.Hour()
		electricityRate := electricityRates[hour]
		integratedRate := electricityRate + baseGridRate + baseRegionalRate + baseMeteringRate

		entry := PriceEntry{
			StartTimestamp: current,
			EndTimestamp:   intervalEnd,
			Electricity: []Rate{
				{Unit: "CHF/kWh", Value: electricityRate},
				{Unit: "CHF/M", Value: monthlyFixed},
			},
			Grid: []Rate{
				{Unit: "CHF/kWh", Value: baseGridRate},
			},
			RegionalFees: []Rate{
				{Unit: "CHF/kWh", Value: baseRegionalRate},
			},
			Metering: []Rate{
				{Unit: "CHF/kWh", Value: baseMeteringRate},
			},
			Integrated: []Rate{
				{Unit: "CHF/kWh", Value: integratedRate},
				{Unit: "CHF/M", Value: monthlyFixed},
			},
		}

		prices = append(prices, entry)
		current = intervalEnd
	}

	return prices
}
