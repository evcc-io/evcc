//go:build integration

package ecoflow

import (
	"os"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

func TestIntegration_MQTTCredentials(t *testing.T) {
	accessKey := os.Getenv("ECOFLOW_ACCESS_KEY")
	secretKey := os.Getenv("ECOFLOW_SECRET_KEY")

	if accessKey == "" || secretKey == "" {
		t.Skip("Skipping: ECOFLOW_ACCESS_KEY and ECOFLOW_SECRET_KEY required")
	}

	uri := os.Getenv("ECOFLOW_URI")
	if uri == "" {
		uri = "https://api-e.ecoflow.com"
	}

	log := util.NewLogger("mqtt-test")
	helper := request.NewHelper(log)
	helper.Client.Transport = NewAuthTransport(helper.Client.Transport, accessKey, secretKey)

	creds, err := GetMQTTCredentials(helper, uri)
	if err != nil {
		t.Fatalf("Failed to get MQTT credentials: %v", err)
	}

	t.Logf("✅ MQTT Credentials:")
	t.Logf("   Account:  %s", creds.Account)
	t.Logf("   Password: %.8s...", creds.Password)
	t.Logf("   Broker:   %s", creds.BrokerURL())

	if creds.Account == "" || creds.Password == "" {
		t.Error("Empty credentials returned")
	}

	if creds.URL == "" || creds.Port == "" {
		t.Error("Empty broker URL/port returned")
	}
}
