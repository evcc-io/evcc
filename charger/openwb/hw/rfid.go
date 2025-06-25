package hw

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/evcc-io/evcc/util"
	"github.com/holoplot/go-evdev"
)

// scanCodeMap maps evdev key codes to their string representations.
var scanCodeMap = map[evdev.EvCode]string{
	evdev.KEY_1:   "1",
	evdev.KEY_2:   "2",
	evdev.KEY_3:   "3",
	evdev.KEY_4:   "4",
	evdev.KEY_5:   "5",
	evdev.KEY_6:   "6",
	evdev.KEY_7:   "7",
	evdev.KEY_8:   "8",
	evdev.KEY_9:   "9",
	evdev.KEY_0:   "0",
	evdev.KEY_KP1: "1",
	evdev.KEY_KP2: "2",
	evdev.KEY_KP3: "3",
	evdev.KEY_KP4: "4",
	evdev.KEY_KP5: "5",
	evdev.KEY_KP6: "6",
	evdev.KEY_KP7: "7",
	evdev.KEY_KP8: "8",
	evdev.KEY_KP9: "9",
	evdev.KEY_KP0: "0",

	// latin letters
	evdev.KEY_A: "A",
	evdev.KEY_B: "B",
	evdev.KEY_C: "C",
	evdev.KEY_D: "D",
	evdev.KEY_E: "E",
	evdev.KEY_F: "F",
	evdev.KEY_G: "G",
	evdev.KEY_H: "H",
	evdev.KEY_I: "I",
	evdev.KEY_J: "J",
	evdev.KEY_K: "K",
	evdev.KEY_L: "L",
	evdev.KEY_M: "M",
	evdev.KEY_N: "N",
	evdev.KEY_O: "O",
	evdev.KEY_P: "P",
	evdev.KEY_Q: "Q",
	evdev.KEY_R: "R",
	evdev.KEY_S: "S",
	evdev.KEY_T: "T",
	evdev.KEY_U: "U",
	evdev.KEY_V: "V",
	evdev.KEY_W: "W",
	evdev.KEY_X: "X",
	evdev.KEY_Y: "Y",
	evdev.KEY_Z: "Z",

	// punctuation marks and other characters
	evdev.KEY_MINUS:      "-",
	evdev.KEY_EQUAL:      "=",
	evdev.KEY_SEMICOLON:  ";",
	evdev.KEY_COMMA:      ",",
	evdev.KEY_DOT:        ".",
	evdev.KEY_SLASH:      "/",
	evdev.KEY_KPASTERISK: "*",
	evdev.KEY_KPMINUS:    "-",
	evdev.KEY_KPPLUS:     "+",
	evdev.KEY_KPDOT:      ".",
	evdev.KEY_KPSLASH:    "/",
}

// NewRFIDHandler initializes RFID device monitoring and returns the channel for RFID reads.
// It also returns a cancel function to stop monitoring and clean up resources.
func NewRFIDHandler(ctx context.Context, log *util.Logger) (chan string, func(), error) {
	devicePaths, err := evdev.ListDevicePaths()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot list device paths: %w", err)
	}

	var keyboardPaths []string
	for _, d := range devicePaths {
		log.INFO.Printf("Device path: %s | Name: %s\n", d.Path, d.Name)
		dev, err := evdev.Open(d.Path)
		if err != nil {
			log.INFO.Printf("Cannot read %s: %v\n", d.Path, err)
			continue
		}
		events := dev.CapableEvents(evdev.EV_KEY)
		if slices.Contains(events, evdev.KEY_ENTER) {
			log.INFO.Println("detected 'enter' key, device seems to be a keyboard")
			keyboardPaths = append(keyboardPaths, d.Path)
		} else {
			log.INFO.Println("no 'enter' key detected, skipping device")
		}
	}

	rfIdChannel := make(chan string, 10)
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	for _, p := range keyboardPaths {
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
		log.INFO.Printf("Cannot read %s: %v\n", p, err)
		return
	}
	var read string
	for {
		select {
		case <-ctx.Done():
			return
		default:
			e, err := dev.ReadOne()
			if err != nil {
				log.INFO.Printf("Error reading from device: %v\n", err)
				continue
			}

			switch e.Type {
			case evdev.EV_KEY:
				if e.Value == 1 {
					log.INFO.Printf("Received keystroke \"%s\"", e.CodeName())
					if e.Code == evdev.KEY_ENTER || e.Code == evdev.KEY_KPENTER {
						rfIdChannel <- read
						read = ""
					} else {
						if val, ok := scanCodeMap[e.Code]; ok {
							read += val
						} else {
							log.INFO.Printf("Unknown key code: %v", e.Code)
						}
					}
				}
			}
		}
	}
}
