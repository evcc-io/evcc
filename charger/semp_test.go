package charger

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock SEMP server responses
const (
	mockDeviceStatusResponse = `<?xml version="1.0" encoding="UTF-8"?>
<Device2EM xmlns="http://www.sma.de/communication/schema/SEMP/v1">
	<DeviceStatus>
		<DeviceId>F-12345678-ABCDEF123456-00</DeviceId>
		<Status>On</Status>
		<EMSignalsAccepted>true</EMSignalsAccepted>
		<PowerConsumption>
			<PowerInfo>
				<AveragePower>7400</AveragePower>
				<Timestamp>1640995200</Timestamp>
				<AveragingInterval>60</AveragingInterval>
			</PowerInfo>
		</PowerConsumption>
	</DeviceStatus>
</Device2EM>`

	mockDeviceStatusOffResponse = `<?xml version="1.0" encoding="UTF-8"?>
<Device2EM xmlns="http://www.sma.de/communication/schema/SEMP/v1">
	<DeviceStatus>
		<DeviceId>F-12345678-ABCDEF123456-00</DeviceId>
		<Status>Off</Status>
		<EMSignalsAccepted>false</EMSignalsAccepted>
		<PowerConsumption>
			<PowerInfo>
				<AveragePower>0</AveragePower>
				<Timestamp>1640995200</Timestamp>
				<AveragingInterval>60</AveragingInterval>
			</PowerInfo>
		</PowerConsumption>
	</DeviceStatus>
</Device2EM>`

	mockDeviceStatusReadyResponse = `<?xml version="1.0" encoding="UTF-8"?>
<Device2EM xmlns="http://www.sma.de/communication/schema/SEMP/v1">
	<DeviceStatus>
		<DeviceId>F-12345678-ABCDEF123456-00</DeviceId>
		<Status>Ready</Status>
		<EMSignalsAccepted>true</EMSignalsAccepted>
		<PowerConsumption>
			<PowerInfo>
				<AveragePower>0</AveragePower>
				<Timestamp>1640995200</Timestamp>
				<AveragingInterval>60</AveragingInterval>
			</PowerInfo>
		</PowerConsumption>
	</DeviceStatus>
</Device2EM>`

	mockPlanningRequestResponse = `<?xml version="1.0" encoding="UTF-8"?>
<Device2EM xmlns="http://www.sma.de/communication/schema/SEMP/v1">
	<PlanningRequest>
		<Timeframe>
			<DeviceId>F-12345678-ABCDEF123456-00</DeviceId>
			<EarliestStart>1640995200</EarliestStart>
			<LatestEnd>1641006000</LatestEnd>
			<MaxEnergy>10000</MaxEnergy>
		</Timeframe>
	</PlanningRequest>
</Device2EM>`

	mockEmptyPlanningRequestResponse = `<?xml version="1.0" encoding="UTF-8"?>
<Device2EM xmlns="http://www.sma.de/communication/schema/SEMP/v1">
</Device2EM>`

	mockDeviceInfoResponse = `<?xml version="1.0" encoding="UTF-8"?>
<Device2EM xmlns="http://www.sma.de/communication/schema/SEMP/v1">
	<DeviceInfo>
		<Identification>
			<DeviceId>F-12345678-ABCDEF123456-00</DeviceId>
			<DeviceName>Test Wallbox</DeviceName>
			<DeviceType>EVCharger</DeviceType>
			<DeviceSerial>123456</DeviceSerial>
			<DeviceVendor>Test Vendor</DeviceVendor>
		</Identification>
		<Characteristics>
			<MinPowerConsumption>0</MinPowerConsumption>
			<MaxPowerConsumption>11000</MaxPowerConsumption>
		</Characteristics>
		<Capabilities>
			<CurrentPower>
				<Method>Measurement</Method>
			</CurrentPower>
			<Timestamps>
				<AbsoluteTimestamps>true</AbsoluteTimestamps>
			</Timestamps>
			<Interruptions>
				<InterruptionsAllowed>true</InterruptionsAllowed>
			</Interruptions>
			<Requests>
				<OptionalEnergy>true</OptionalEnergy>
			</Requests>
		</Capabilities>
	</DeviceInfo>
</Device2EM>`

	mockDeviceInfoNoInterruptionsResponse = `<?xml version="1.0" encoding="UTF-8"?>
<Device2EM xmlns="http://www.sma.de/communication/schema/SEMP/v1">
	<DeviceInfo>
		<Identification>
			<DeviceId>F-12345678-ABCDEF123456-00</DeviceId>
			<DeviceName>Test Wallbox</DeviceName>
			<DeviceType>EVCharger</DeviceType>
			<DeviceSerial>123456</DeviceSerial>
			<DeviceVendor>Test Vendor</DeviceVendor>
		</Identification>
		<Characteristics>
			<MinPowerConsumption>0</MinPowerConsumption>
			<MaxPowerConsumption>11000</MaxPowerConsumption>
		</Characteristics>
		<Capabilities>
			<CurrentPower>
				<Method>Measurement</Method>
			</CurrentPower>
			<Timestamps>
				<AbsoluteTimestamps>true</AbsoluteTimestamps>
			</Timestamps>
			<Interruptions>
				<InterruptionsAllowed>false</InterruptionsAllowed>
			</Interruptions>
			<Requests>
				<OptionalEnergy>true</OptionalEnergy>
			</Requests>
		</Capabilities>
	</DeviceInfo>
</Device2EM>`

	mockDeviceInfoPhases1p3pResponse = `<?xml version="1.0" encoding="UTF-8"?>
<Device2EM xmlns="http://www.sma.de/communication/schema/SEMP/v1">
	<DeviceInfo>
		<Identification>
			<DeviceId>F-12345678-ABCDEF123456-00</DeviceId>
			<DeviceName>Test Wallbox</DeviceName>
			<DeviceType>EVCharger</DeviceType>
			<DeviceSerial>123456</DeviceSerial>
			<DeviceVendor>Test Vendor</DeviceVendor>
		</Identification>
		<Characteristics>
			<MinPowerConsumption>3600</MinPowerConsumption>
			<MaxPowerConsumption>11000</MaxPowerConsumption>
		</Characteristics>
		<Capabilities>
			<CurrentPower>
				<Method>Measurement</Method>
			</CurrentPower>
			<Timestamps>
				<AbsoluteTimestamps>true</AbsoluteTimestamps>
			</Timestamps>
			<Interruptions>
				<InterruptionsAllowed>true</InterruptionsAllowed>
			</Interruptions>
			<Requests>
				<OptionalEnergy>true</OptionalEnergy>
			</Requests>
		</Capabilities>
	</DeviceInfo>
</Device2EM>`
)

type sempTestHandler struct {
	statusResponse   string
	planningResponse string
	infoResponse     string
	lastRequest      string
	requestCount     int
}

func (h *sempTestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.requestCount++

	// Handle POST to base URL for DeviceControl
	if r.Method == http.MethodPost {
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		h.lastRequest = string(body)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Handle GET requests
	switch r.URL.Path {
	case "/semp":
		// Return complete SEMP document at base URL
		// Extract and combine the XML fragments
		var parts []string

		// Add DeviceInfo
		if start := strings.Index(h.infoResponse, "<DeviceInfo>"); start != -1 {
			if end := strings.Index(h.infoResponse, "</Device2EM>"); end != -1 {
				parts = append(parts, h.infoResponse[start:end])
			}
		}

		// Add DeviceStatus
		if start := strings.Index(h.statusResponse, "<DeviceStatus>"); start != -1 {
			if end := strings.Index(h.statusResponse, "</Device2EM>"); end != -1 {
				parts = append(parts, h.statusResponse[start:end])
			}
		}

		// Add PlanningRequest if exists
		if start := strings.Index(h.planningResponse, "<PlanningRequest>"); start != -1 {
			if end := strings.Index(h.planningResponse, "</Device2EM>"); end != -1 {
				parts = append(parts, h.planningResponse[start:end])
			}
		}

		fullDoc := `<?xml version="1.0" encoding="UTF-8"?>
<Device2EM xmlns="http://www.sma.de/communication/schema/SEMP/v1">` +
			strings.Join(parts, "") +
			`</Device2EM>`
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(fullDoc))

	case "/semp/DeviceStatus":
		// Legacy endpoint - still supported
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(h.statusResponse))

	case "/semp/DeviceInfo":
		// Legacy endpoint - still supported
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(h.infoResponse))

	case "/semp/PlanningRequest":
		// Legacy endpoint - still supported
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(h.planningResponse))

	default:
		http.NotFound(w, r)
	}
}

func TestSEMPCharger(t *testing.T) {
	handler := &sempTestHandler{
		statusResponse:   mockDeviceStatusResponse,
		planningResponse: mockPlanningRequestResponse,
		infoResponse:     mockDeviceInfoResponse,
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	wb, err := NewSEMP(server.URL+"/semp", "F-12345678-ABCDEF123456-00", time.Second)
	require.NoError(t, err)

	t.Run("Status", func(t *testing.T) {
		status, err := wb.Status()
		require.NoError(t, err)
		assert.Equal(t, api.StatusC, status)
		assert.Equal(t, 1, handler.requestCount) // Only 1 request to base URL for full document (DeviceInfo in NewSEMP was cached)
	})

	t.Run("Enabled", func(t *testing.T) {
		handler.requestCount = 0
		// Reset cache to force new request
		wb.(*SEMP).documentG.Reset()
		enabled, err := wb.Enabled()
		require.NoError(t, err)
		assert.True(t, enabled)
		assert.Equal(t, 1, handler.requestCount) // Only DeviceStatus for Enabled
	})

	t.Run("CurrentPower", func(t *testing.T) {
		handler.requestCount = 0
		// Reset cache to force new request
		wb.(*SEMP).documentG.Reset()
		meter, ok := wb.(api.Meter)
		require.True(t, ok)
		power, err := meter.CurrentPower()
		require.NoError(t, err)
		assert.Equal(t, 7400.0, power)
		assert.Equal(t, 1, handler.requestCount) // Only DeviceStatus for CurrentPower
	})

	t.Run("Enable", func(t *testing.T) {
		handler.requestCount = 0
		// Reset caches to ensure fresh requests
		wb.(*SEMP).documentG.Reset()
		err := wb.Enable(true)
		require.NoError(t, err)
		assert.Contains(t, handler.lastRequest, "<On>true</On>")
		assert.Contains(t, handler.lastRequest, "F-12345678-ABCDEF123456-00")
		// calcPower() returns 0 because current is not set yet
		assert.Contains(t, handler.lastRequest, "<RecommendedPowerConsumption>0</RecommendedPowerConsumption>")
		assert.Equal(t, 2, handler.requestCount) // 1 GET for full document + 1 POST for DeviceControl
	})

	t.Run("MaxCurrent", func(t *testing.T) {
		handler.requestCount = 0
		err := wb.MaxCurrent(16)
		require.NoError(t, err)
		// calcPower() = 230 * 3 (phases) * 16 (current) = 11040 (enabled from previous test)
		assert.Contains(t, handler.lastRequest, "<RecommendedPowerConsumption>11040</RecommendedPowerConsumption>")
		assert.Equal(t, 1, handler.requestCount) // Only DeviceControl
	})
}

func TestSEMPChargerOff(t *testing.T) {
	handler := &sempTestHandler{
		statusResponse:   mockDeviceStatusOffResponse,
		planningResponse: mockEmptyPlanningRequestResponse,
		infoResponse:     mockDeviceInfoResponse,
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	wb, err := NewSEMP(server.URL+"/semp", "F-12345678-ABCDEF123456-00", time.Second)
	require.NoError(t, err)

	t.Run("StatusOff", func(t *testing.T) {
		status, err := wb.Status()
		require.NoError(t, err)
		assert.Equal(t, api.StatusA, status)
	})

	t.Run("DisabledWhenOff", func(t *testing.T) {
		enabled, err := wb.Enabled()
		require.NoError(t, err)
		assert.False(t, enabled)
	})

	t.Run("CurrentPowerZero", func(t *testing.T) {
		meter, ok := wb.(api.Meter)
		require.True(t, ok)
		power, err := meter.CurrentPower()
		require.NoError(t, err)
		assert.Equal(t, 0.0, power)
	})
}

func TestSEMPChargerDeviceNotFound(t *testing.T) {
	handler := &sempTestHandler{
		statusResponse:   strings.Replace(mockDeviceStatusResponse, "F-12345678-ABCDEF123456-00", "DIFFERENT-DEVICE-ID", -1),
		planningResponse: mockPlanningRequestResponse,
		infoResponse:     mockDeviceInfoResponse,
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	// NewSEMP now calls Enabled() which will fail if device is not found
	_, err := NewSEMP(server.URL+"/semp", "F-12345678-ABCDEF123456-00", time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "device F-12345678-ABCDEF123456-00 not found")
}

func TestSEMPChargerReady(t *testing.T) {
	handler := &sempTestHandler{
		statusResponse:   mockDeviceStatusReadyResponse,
		planningResponse: mockPlanningRequestResponse,
		infoResponse:     mockDeviceInfoResponse,
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	wb, err := NewSEMP(server.URL+"/semp", "F-12345678-ABCDEF123456-00", time.Second)
	require.NoError(t, err)

	t.Run("StatusReady", func(t *testing.T) {
		status, err := wb.Status()
		require.NoError(t, err)
		assert.Equal(t, api.StatusB, status) // EMSignalsAccepted=true but Status!=On -> Status B
	})

	t.Run("EnabledWhenReady", func(t *testing.T) {
		enabled, err := wb.Enabled()
		require.NoError(t, err)
		assert.False(t, enabled) // Status is not "On" so not enabled
	})
}

func TestSEMPChargerNoInterruptions(t *testing.T) {
	handler := &sempTestHandler{
		statusResponse:   mockDeviceStatusResponse,
		planningResponse: mockPlanningRequestResponse,
		infoResponse:     mockDeviceInfoNoInterruptionsResponse,
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	wb, err := NewSEMP(server.URL+"/semp", "F-12345678-ABCDEF123456-00", time.Second)
	require.NoError(t, err)

	t.Run("EnabledReturnsFalseWhenInterruptionsNotAllowed", func(t *testing.T) {
		// Enabled() no longer checks InterruptionsAllowed, only status
		enabled, err := wb.Enabled()
		require.NoError(t, err)
		assert.True(t, enabled) // Status is "On" and EMSignalsAccepted is true
	})

	t.Run("EnableReturnsErrorWhenInterruptionsNotAllowed", func(t *testing.T) {
		err := wb.Enable(true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "device does not allow control")
	})

	t.Run("MaxCurrentSucceedsWhenInterruptionsNotAllowed", func(t *testing.T) {
		// MaxCurrent calls MaxCurrentMillis which doesn't check InterruptionsAllowed
		err := wb.MaxCurrent(16)
		require.NoError(t, err)
		// calcPower() uses enabled state which was initialized from device status (true)
		// Power = 230V * 3 phases * 16A = 11040W
		assert.Contains(t, handler.lastRequest, "<On>true</On>")
		assert.Contains(t, handler.lastRequest, "<RecommendedPowerConsumption>11040</RecommendedPowerConsumption>")
	})

	t.Run("MaxCurrentMillisSucceedsWhenInterruptionsNotAllowed", func(t *testing.T) {
		chargerEx, ok := wb.(api.ChargerEx)
		require.True(t, ok)
		err := chargerEx.MaxCurrentMillis(16.0)
		require.NoError(t, err)
		// calcPower() uses enabled state which was initialized from device status (true)
		// Power = 230V * 3 phases * 16A = 11040W
		assert.Contains(t, handler.lastRequest, "<On>true</On>")
		assert.Contains(t, handler.lastRequest, "<RecommendedPowerConsumption>11040</RecommendedPowerConsumption>")
	})
}

func TestSEMPChargerPhases1p3p(t *testing.T) {
	handler := &sempTestHandler{
		statusResponse:   mockDeviceStatusResponse,
		planningResponse: mockPlanningRequestResponse,
		infoResponse:     mockDeviceInfoPhases1p3pResponse,
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	wb, err := NewSEMP(server.URL+"/semp", "F-12345678-ABCDEF123456-00", time.Second)
	require.NoError(t, err)

	// Check if the charger supports phase switching
	phaseSwitcher, ok := wb.(api.PhaseSwitcher)
	require.True(t, ok, "Expected charger to support phase switching")

	t.Run("SwitchTo1Phase", func(t *testing.T) {
		handler.requestCount = 0
		// Enable the charger first so calcPower() returns a non-zero value
		err := wb.Enable(true)
		require.NoError(t, err)
		// Set current to have a predictable power calculation
		err = wb.MaxCurrent(16)
		require.NoError(t, err)
		err = phaseSwitcher.Phases1p3p(1)
		require.NoError(t, err)
		// calcPower() = 230 * 1 * 16 = 3680
		assert.Contains(t, handler.lastRequest, "<RecommendedPowerConsumption>3680</RecommendedPowerConsumption>")
		assert.Equal(t, 5, handler.requestCount) // Enable (1 GET + 1 POST) + MaxCurrent (1 POST) + Phases1p3p (2 POST: stop + restart)
	})

	t.Run("SwitchTo3Phase", func(t *testing.T) {
		handler.requestCount = 0
		err := phaseSwitcher.Phases1p3p(3)
		require.NoError(t, err)
		// calcPower() = 230 * 3 * 16 = 11040 (enabled=true and current=16 from previous test)
		assert.Contains(t, handler.lastRequest, "<RecommendedPowerConsumption>11040</RecommendedPowerConsumption>")
		assert.Equal(t, 3, handler.requestCount) // 1 GET (DeviceInfo) + 2 POST (stop + restart DeviceControl)
	})
}
