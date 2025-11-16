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

type RfIdContainer struct {
	mu   sync.Mutex
	rfId string
}

func (c *RfIdContainer) Set(rfId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rfId = rfId
}

func (c *RfIdContainer) Get() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.rfId
}

// NewRFIDHandler initializes RFID device monitoring.
// It starts goroutines that monitor RFID devices and update the rfIdContainer.
// Monitoring stops when the context is cancelled.
func NewRFIDHandler(rfIdVidPid string, ctx context.Context, rfIdContainer *RfIdContainer, log *util.Logger) error {
	devicePaths, err := evdev.ListDevicePaths()
	if err != nil {
		return fmt.Errorf("cannot list device paths: %w", err)
	}
	var keyboardPaths []string
	for _, d := range devicePaths {
		log.DEBUG.Printf("Device path: %s | Name: %s\n", d.Path, d.Name)
		dev, err := evdev.Open(d.Path)
		if err != nil {
			log.WARN.Printf("Cannot read %s: %v\n", d.Path, err)
			continue
		}
		inputId, err := dev.InputID()
		if err != nil {
			log.WARN.Printf("Cannot get InputID for %s: %s", d.Path, err)
			continue
		}
		if fmt.Sprintf("%x:%x", inputId.Vendor, inputId.Product) == rfIdVidPid {
			log.DEBUG.Printf("found input device which matches VID:PID %s", rfIdVidPid)
			events := dev.CapableEvents(evdev.EV_KEY)
			if slices.Contains(events, evdev.KEY_ENTER) {
				log.DEBUG.Println("detected 'enter' key, device seems to be a keyboard")
				keyboardPaths = append(keyboardPaths, d.Path)
			} else {
				log.DEBUG.Println("no 'enter' key detected, skipping device")
			}
		}
	}
	for _, p := range keyboardPaths {
		go monitorKeyboardRFID(ctx, p, log, rfIdContainer)
	}
	return nil
}

// monitorKeyboardRFID listens for RFID input events from the specified device path `p`
// and sends complete RFID reads to the `rfIdChannel` channel.
// It stops when the context is cancelled and signals completion via the WaitGroup.
func monitorKeyboardRFID(ctx context.Context, p string, log *util.Logger, rfIdContainer *RfIdContainer) {
	dev, err := evdev.Open(p)
	if err != nil {
		log.ERROR.Printf("Cannot read %s: %v\n", p, err)
		return
	}

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
						log.DEBUG.Printf("Received enter key, setting RFID \"%s\"", builder.String())
						rfIdContainer.Set(builder.String())
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
