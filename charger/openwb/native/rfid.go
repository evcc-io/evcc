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

// NewRFIDHandler initializes RFID device monitoring and returns the channel for RFID reads.
// It also returns a cancel function to stop monitoring and clean up resources.
func NewRFIDHandler(ctx context.Context, log *util.Logger) (chan string, func(), error) {
	devicePaths, err := evdev.ListDevicePaths()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot list device paths: %w", err)
	}

	var keyboardPaths []string
	for _, d := range devicePaths {
		log.DEBUG.Printf("Device path: %s | Name: %s\n", d.Path, d.Name)
		dev, err := evdev.Open(d.Path)
		if err != nil {
			log.WARN.Printf("Cannot read %s: %v\n", d.Path, err)
			continue
		}
		events := dev.CapableEvents(evdev.EV_KEY)
		if slices.Contains(events, evdev.KEY_ENTER) {
			log.DEBUG.Println("detected 'enter' key, device seems to be a keyboard")
			keyboardPaths = append(keyboardPaths, d.Path)
		} else {
			log.DEBUG.Println("no 'enter' key detected, skipping device")
		}
	}

	rfIdChannel := make(chan string, 10)
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	for _, p := range keyboardPaths {
		wg.Go(func() {
			monitorKeyboardRFID(ctx, p, log, rfIdChannel)
		})
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

			var builder strings.Builder

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
	} else if after, found := strings.CutPrefix(s, "KEY_"); found && len(after) == 1 { // Events from regular keys
		return after, true
	}
	return "", false // Unknown key
}
