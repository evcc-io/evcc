package charger

import (
	"context"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIAquaLinkFromConfig(t *testing.T) {
	tests := []struct {
	name    string
	config  map[string]interface{}
	wantErr bool
	errMsg  string
	}{
		{
			name: "valid local mode config",
			config: map[string]interface{}{
				"uri": "http://192.168.1.100",
			},
			wantErr: false,
		},
		{
			name: "valid cloud mode config",
			config: map[string]interface{}{
				"email":    "test@example.com",
				"password": "password123",
				"device":   "device123",
			},
			wantErr: false,
		},
		{
			name: "missing both uri and credentials",
			config: map[string]interface{}{
				"device": "device123",
			},
			wantErr: true,
			errMsg:  "must provide either uri (local mode) or email/password (cloud mode)",
		},
		{
			name: "missing email in cloud mode",
			config: map[string]interface{}{
				"password": "password123",
				"device":   "device123",
			},
			wantErr: true,
			errMsg:  "must provide either uri (local mode) or email/password (cloud mode)",
		},
		{
			name: "missing password in cloud mode",
			config: map[string]interface{}{
				"email":  "test@example.com",
				"device": "device123",
			},
			wantErr: true,
			errMsg:  "must provide either uri (local mode) or email/password (cloud mode)",
		},
		{
			name: "missing device in cloud mode",
			config: map[string]interface{}{
				"email":    "test@example.com",
				"password": "password123",
			},
			wantErr: true,
			errMsg:  "device name is required for cloud mode",
		},
		{
			name: "both uri and credentials provided",
			config: map[string]interface{}{
				"uri":      "http://192.168.1.100",
				"email":    "test@example.com",
				"password": "password123",
				"device":   "device123",
			},
			wantErr: true,
			errMsg:  "cannot use both uri (local) and email/password (cloud) - choose one mode",
		},
		{
			name: "uri with email only",
			config: map[string]interface{}{
				"uri":   "http://192.168.1.100",
				"email": "test@example.com",
			},
			wantErr: true,
			errMsg:  "cannot use both uri (local) and email/password (cloud) - choose one mode",
		},
		{
			name: "uri with password only",
			config: map[string]interface{}{
				"uri":      "http://192.168.1.100",
				"password": "password123",
			},
			wantErr: true,
			errMsg:  "cannot use both uri (local) and email/password (cloud) - choose one mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			charger, err := NewIAquaLinkFromConfig(ctx, tt.config)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, charger)
			} else {
				// For valid configs, we expect an error during actual initialization
				// (since we don't have real credentials/devices), but the config validation should pass
				// The error should be from the actual device connection, not config validation
				if err != nil {
					// Config validation passed, but device connection failed (expected in tests)
					// This is acceptable - we're testing config validation, not device connectivity
					assert.NotContains(t, err.Error(), "must provide either uri")
					assert.NotContains(t, err.Error(), "cannot use both uri")
					assert.NotContains(t, err.Error(), "device name is required")
				} else {
					// If no error, verify it implements the required interface
					require.NotNil(t, charger)
					_, ok := charger.(api.Charger)
					assert.True(t, ok, "charger should implement api.Charger")
					_, ok = charger.(api.ChargerEx)
					assert.True(t, ok, "charger should implement api.ChargerEx")
				}
			}
		})
	}
}

func TestIAquaLink_InterfaceImplementation(t *testing.T) {
	// Test that IAquaLink properly implements required interfaces
	// This is a compile-time check, but we can also verify at runtime
	ctx := context.Background()
	config := map[string]interface{}{
		"uri": "http://192.168.1.100",
	}

	charger, err := NewIAquaLinkFromConfig(ctx, config)
	if err != nil {
		// If initialization fails due to network/device issues, that's acceptable
		// We're just checking that the type is correct
		t.Skipf("Skipping interface test due to initialization error: %v", err)
		return
	}

	// Verify it implements api.Charger
	_, ok := charger.(api.Charger)
	assert.True(t, ok, "IAquaLink should implement api.Charger")

	// Verify it implements api.ChargerEx (via SgReady)
	_, ok = charger.(api.ChargerEx)
	assert.True(t, ok, "IAquaLink should implement api.ChargerEx via SgReady")

	// Verify it implements api.Dimmer (via SgReady)
	_, ok = charger.(api.Dimmer)
	assert.True(t, ok, "IAquaLink should implement api.Dimmer via SgReady")
}

