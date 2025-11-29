package battery

import (
	"context"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/libp2p/zeroconf/v2"
)

// DeviceType represents the type of HomeWizard device
type DeviceType string

const (
	DeviceTypeP1Meter DeviceType = "p1meter"
	DeviceTypeBattery DeviceType = "battery"
	DeviceTypeUnknown DeviceType = "unknown"
)

// DiscoveredDevice represents a discovered HomeWizard device
type DiscoveredDevice struct {
	Instance string
	Host     string
	Port     int
	Type     DeviceType
}

// DiscoverDevices scans the network for HomeWizard devices (P1 meters and batteries)
// Calls onDevice for each discovered device. Returns when context is cancelled or timeout expires.
func DiscoverDevices(ctx context.Context, onDevice func(DiscoveredDevice)) error {
	log := util.NewLogger("homewizard")
	log.DEBUG.Printf("starting mDNS discovery for _homewizard._tcp")

	entries := make(chan *zeroconf.ServiceEntry, 10)

	// Collect entries in a goroutine
	go func() {
		for entry := range entries {
			// Log raw DNS record details
			log.TRACE.Printf("mDNS entry: Instance=%s, HostName=%s, Port=%d, AddrIPv4=%v, AddrIPv6=%v, Text=%v",
				entry.Instance, entry.HostName, entry.Port, entry.AddrIPv4, entry.AddrIPv6, entry.Text)

			// Extract product_type from TXT records
			productType := extractProductType(entry.Text)
			if productType == "" {
				log.DEBUG.Printf("skipping device %s: no product_type in TXT records", entry.Instance)
				continue
			}

			// Determine device type from product_type TXT record
			var deviceType DeviceType
			switch productType {
			case "HWE-P1":
				deviceType = DeviceTypeP1Meter
			case "HWE-BAT":
				deviceType = DeviceTypeBattery
			default:
				// Skip unknown product types
				log.DEBUG.Printf("skipping device %s: unknown product_type=%s", entry.Instance, productType)
				continue
			}

			// Extract resolvable hostname (removing .local. suffix if present)
			host := strings.TrimSuffix(entry.HostName, ".local.")

			device := DiscoveredDevice{
				Instance: entry.Instance,
				Host:     host,
				Port:     entry.Port,
				Type:     deviceType,
			}

			log.DEBUG.Printf("discovered %s: %s at %s:%d", deviceType, entry.Instance, host, entry.Port)
			onDevice(device)
		}
	}()

	// Browse for HomeWizard devices using the _homewizard._tcp service
	// The entries channel will be closed by zeroconf when done
	if err := zeroconf.Browse(ctx, "_homewizard._tcp", "local.", entries); err != nil {
		return fmt.Errorf("failed to browse for HomeWizard devices: %w", err)
	}

	<-ctx.Done()
	return nil
}

// extractProductType parses TXT records to find the product_type field
// TXT records are in format "key=value", e.g., "product_type=HWE-P1"
func extractProductType(txtRecords []string) string {
	for _, txt := range txtRecords {
		if key, value, found := strings.Cut(txt, "="); found && key == "product_type" {
			return value
		}
	}
	return ""
}
