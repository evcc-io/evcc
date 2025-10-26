package native

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/util"
	"github.com/holoplot/go-evdev"
)

// isLikelyRFIDReader checks if a device is likely an RFID reader based on its capabilities.
// RFID readers typically only have numeric keys (0-9) and ENTER, while real keyboards
// have many more keys like letters, function keys, etc.
// Some RFID readers also output hex values (A-F) and hyphens.
func isLikelyRFIDReader(dev *evdev.InputDevice) bool {
	events := dev.CapableEvents(evdev.EV_KEY)

	// Must have ENTER key
	if !slices.Contains(events, evdev.KEY_ENTER) && !slices.Contains(events, evdev.KEY_KPENTER) {
		return false
	}

	// Count different key categories
	hasNonHexLetters := false // G-Z (indicates real keyboard)
	hasFunctionKeys := false
	hasNumericKeys := false
	keyCount := 0

	for _, event := range events {
		keyCount++

		// Check for hex letter keys (A-F) - allowed for RFID readers
		// (no action needed, just skip them)

		// Check for non-hex letter keys (G-Z)
		if event >= evdev.KEY_G && event <= evdev.KEY_Z {
			hasNonHexLetters = true
		}
		// Check for function keys (F1-F12)
		if event >= evdev.KEY_F1 && event <= evdev.KEY_F12 {
			hasFunctionKeys = true
		}
		// Check for numeric keys (0-9 or numpad)
		if (event >= evdev.KEY_0 && event <= evdev.KEY_9) ||
			(event >= evdev.KEY_KP0 && event <= evdev.KEY_KP9) {
			hasNumericKeys = true
		}
	}

	// RFID reader characteristics:
	// - Has numeric keys and ENTER
	// - May have A-F for hex output
	// - Does NOT have G-Z or function keys
	// - Has relatively few keys overall (< 30 is a good threshold)
	return hasNumericKeys && !hasNonHexLetters && !hasFunctionKeys && keyCount < 30
}

// hasRFIDLikeName checks if a device name suggests it's an RFID reader.
func hasRFIDLikeName(name string) bool {
	rfidKeywords := []string{"rfid", "card reader", "barcode", "scanner", "mifare", "nfc"}
	keyboardKeywords := []string{"keyboard", "tastatur"}

	nameLower := strings.ToLower(name)

	// Explicitly identified as keyboard -> exclude
	for _, kw := range keyboardKeywords {
		if strings.Contains(nameLower, kw) {
			return false
		}
	}

	// RFID-typical keywords
	for _, kw := range rfidKeywords {
		if strings.Contains(nameLower, kw) {
			return true
		}
	}

	return false
}

// NewRFIDHandler initializes RFID device monitoring and returns the channel for RFID reads.
// It also returns a cancel function to stop monitoring and clean up resources.
func NewRFIDHandler(ctx context.Context, log *util.Logger) (chan string, func(), error) {
	devicePaths, err := evdev.ListDevicePaths()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot list device paths: %w", err)
	}

	var rfidPaths []string
	for _, d := range devicePaths {
		log.DEBUG.Printf("Device path: %s | Name: %s\n", d.Path, d.Name)
		dev, err := evdev.Open(d.Path)
		if err != nil {
			log.WARN.Printf("Cannot read %s: %v\n", d.Path, err)
			continue
		}

		// Multi-stage detection: capabilities + name heuristic
		isRFIDByCapabilities := isLikelyRFIDReader(dev)
		isRFIDByName := hasRFIDLikeName(d.Name)

		if isRFIDByCapabilities || isRFIDByName {
			log.DEBUG.Printf("Device identified as RFID reader (capabilities: %v, name: %v)", isRFIDByCapabilities, isRFIDByName)
			rfidPaths = append(rfidPaths, d.Path)
		} else {
			log.DEBUG.Println("Device does not match RFID reader criteria, skipping")
		}

		dev.Close()
	}

	rfIdChannel := make(chan string, 10)
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	for _, p := range rfidPaths {
		wg.Add(1)
		go monitorKeyboardRFID(ctx, p, log, rfIdChannel, &wg)
	}
	cleanup := func() {
		cancel()
		wg.Wait()
		close(rfIdChannel)
	}
	return rfIdChannel, cleanup, nil
}

// monitorKeyboardRFID listens for RFID input events from the specified device path `p`
// and sends complete RFID reads to the `rfIdChannel` channel.
// It stops when the context is cancelled and signals completion via the WaitGroup.
func monitorKeyboardRFID(ctx context.Context, p string, log *util.Logger, rfIdChannel chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	dev, err := evdev.Open(p)
	if err != nil {
		log.ERROR.Printf("Cannot read %s: %v\n", p, err)
		return
	}
	defer dev.Close()

	var builder strings.Builder

	for {
		select {
		case <-ctx.Done():
			return
		default:
			e, err := dev.ReadOne()
			if err != nil {
				log.ERROR.Printf("Error reading from device: %v\n", err)
				continue
			}

			switch e.Type {
			case evdev.EV_KEY:
				if e.Value == 1 {
					log.DEBUG.Printf("Received keystroke \"%s\"", e.CodeName())
					if e.Code == evdev.KEY_ENTER || e.Code == evdev.KEY_KPENTER {
						rfIdChannel <- builder.String()
						builder.Reset()
					} else {
						if val, ok := convertKeyCodeNameToCharacter(e.CodeName()); ok {
							builder.WriteString(val)
						} else {
							log.WARN.Printf("Unknown key code: %v", e.Code)
						}
					}
				}
			}
		}
	}
}

func convertKeyCodeNameToCharacter(s string) (string, bool) {
	if after, found := strings.CutPrefix(s, "KEY_KP"); found && len(after) == 1 { // Events from numeric keypad
		return after, true
	} else if after, found := strings.CutPrefix(s, "KEY_"); found && len(after) == 1 { // Events from regular keys (0-9, A-F for hex)
		return after, true
	}
	return "", false // Unknown key
}
