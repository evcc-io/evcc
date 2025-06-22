package charger

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/holoplot/go-evdev"
	"github.com/stianeikeland/go-rpio/v4"
)

// OpenWbNative charger implementation
type OpenWbNative struct {
	conn        *modbus.Connection
	current     int64
	log         *util.Logger
	rfIdChannel *chan string
	rfId        string
}

const (
	openwbnativeRegAmpsConfig    = 1000
	openwbnativeRegVehicleStatus = 1002
)

func init() {
	registry.AddCtx("openwbnative", NewOpenWbNativeFromConfig)
}

var scan_code_map = map[evdev.EvCode]string{
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

//     GPIO 5 => 1 phasig, Schütz A an, Schütz B aus.
//    GPIO 26 => 3 phasig, Schütz A und B an.
//    Die CP-Unterbrechung wird über ein normales Relais mit NC auf BCM 25/Board 22 gesteuert.
//gpio=4,5,7,11,17,22,23,24,25,26,27=op,dl
//gpio=6,8,9,10,12,13,16,21=ip,pu

//go:generate go tool decorate -f decorateOpenWbNative -b *OpenWbNative -r api.Charger -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.Identifier,Identify,func() (string, error)"

// NewOpenWbNativeFromConfig creates an OpenWbNative DIN charger from generic config
func NewOpenWbNativeFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Phases1p3p      bool
		RfId            bool
		modbus.Settings `mapstructure:",squash"`
	}{
		Settings: modbus.Settings{
			Baudrate: 9600,
			Comset:   "8N1",
			ID:       1},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewOpenWbNative(ctx, cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.Protocol(), cc.ID)
	if err != nil {
		return nil, err
	}

	var phases1p3p func(int) error
	if cc.Phases1p3p {
		phases1p3p = wb.phases1p3p
	}

	if cc.RfId {
		log := util.NewLogger("openwbnative")
		log.INFO.Println("Trying to find RFID device")
		devicePaths, err := evdev.ListDevicePaths()
		if err != nil {
			fmt.Printf("Cannot list device paths: %s", err)
			return nil, err
		}
		var keyboardPaths []string
		for _, d := range devicePaths {
			log.INFO.Printf("%s:\t%s\n", d.Path, d.Name)
			dev, err := evdev.Open(d.Path)
			if err != nil {
				log.INFO.Printf("Cannot read %s: %v\n", d.Path, err)
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
		wb.rfIdChannel = &rfIdChannel
		for _, p := range keyboardPaths {
			go func(p string) {
				log.INFO.Printf("Monitoring keyboard %s\n", p)
				dev, err := evdev.Open(p)
				if err != nil {
					log.INFO.Printf("Cannot read %s: %v\n", p, err)
				}
				var read string = ""
				for {
					e, err := dev.ReadOne()
					if err != nil {
						log.INFO.Printf("Error reading from device: %v\n", err)
					}
					log.INFO.Printf("Got event \"%s\"", e.String())
					switch e.Type {
					case evdev.EV_KEY:
						if e.Value == 1 {
							log.INFO.Printf("Received keystroke \"%s\"", e.CodeName())
							if e.Code == evdev.KEY_ENTER || e.Code == evdev.KEY_KPENTER {
								log.INFO.Printf("Complete rfid \"%s\"", read)
								rfIdChannel <- read
								read = ""
							} else {
								log.INFO.Printf("Adding key \"%s\"", scan_code_map[e.Code])
								read += scan_code_map[e.Code]
							}
						}
					}
				}
			}(p)
		}
	}

	var identify func() (string, error)
	if _, err := wb.identify(); err == nil {
		identify = wb.identify
	}

	return decorateOpenWbNative(wb, phases1p3p, identify), nil
}

// NewOpenWbNative creates OpenWbNative charger
func NewOpenWbNative(ctx context.Context, uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8) (*OpenWbNative, error) {
	log := util.NewLogger("openwbnative")

	conn, err := modbus.NewConnection(ctx, uri, device, comset, baudrate, proto, slaveID)
	if err != nil {
		return nil, err
	}

	conn.Logger(log.TRACE)
	conn.Delay(200 * time.Millisecond)

	openwbnative := &OpenWbNative{
		conn:    conn,
		current: 6, // assume min current
		log:     log,
	}

	openwbnative.log.INFO.Println("OpenWbNative Instantiated...")

	return openwbnative, nil
}

// Status implements the api.Charger interface
func (openwbnative *OpenWbNative) Status() (api.ChargeStatus, error) {
	b, err := openwbnative.conn.ReadHoldingRegisters(openwbnativeRegVehicleStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}
	openwbnative.log.INFO.Printf("Read status %d", b[1])
	switch b[1] {
	case 1: // ready
		return api.StatusA, nil
	case 2: // EV is present
		return api.StatusB, nil
	case 3: // charging
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", b[1])
	}
}

// Enabled implements the api.Charger interface
func (openwbnative *OpenWbNative) Enabled() (bool, error) {
	b, err := openwbnative.conn.ReadHoldingRegisters(openwbnativeRegAmpsConfig, 1)
	if err != nil {
		return false, err
	}

	enabled := b[1] != 0
	if enabled {
		openwbnative.current = int64(b[1])
	}
	openwbnative.log.INFO.Printf("IsEnabled %t", enabled)
	return enabled, nil
}

// Enable implements the api.Charger interface
func (openwbnative *OpenWbNative) Enable(enable bool) error {
	b := []byte{0, 0}

	if enable {
		b[1] = byte(openwbnative.current)
	}

	openwbnative.log.INFO.Printf("Enable charger %t", enable)

	_, err := openwbnative.conn.WriteMultipleRegisters(openwbnativeRegAmpsConfig, 1, b)

	return err
}

// MaxCurrent implements the api.Charger interface
func (openwbnative *OpenWbNative) MaxCurrent(current int64) error {
	b := []byte{0, byte(current)}
	openwbnative.log.INFO.Printf("Set MaxCurrent %d", current)
	_, err := openwbnative.conn.WriteMultipleRegisters(openwbnativeRegAmpsConfig, 1, b)
	if err == nil {
		openwbnative.current = current
	}

	return err
}

// phases1p3p implements the api.PhaseSwitcher interface
func (openwbnative *OpenWbNative) phases1p3p(phases int) error {
	openwbnative.log.INFO.Println("Initiating phase switch...")

	return openwbnative.perform_phase_switch(phases, 60)
}

var _ api.Resurrector = (*OpenWbNative)(nil)

// WakeUp implements the api.Resurrector interface
func (openwbnative *OpenWbNative) WakeUp() error {
	openwbnative.log.INFO.Println("Triggering CP...")
	return openwbnative.perform_cp_interruption(5)
}

// Identify implements the api.Identifier interface
func (openwbnative *OpenWbNative) identify() (string, error) {
	openwbnative.log.INFO.Println("Reading RFID...")
	var completed bool = false
	for !completed {
		select {
		case rfid := <-*openwbnative.rfIdChannel:
			openwbnative.log.INFO.Printf("Read RFID \"%s\" from channel", rfid)
			openwbnative.rfId = rfid
		default:
			openwbnative.log.INFO.Println("Nothing left to read from channel")
			completed = true
		}
	}
	openwbnative.log.INFO.Printf("Returning RFID \"%s\"", openwbnative.rfId)
	return openwbnative.rfId, nil
}

func (openwbnative *OpenWbNative) perform_phase_switch(phases_to_use int, seconds int) error {
	gpio_cp, gpio_relay := get_pins_phase_switch(phases_to_use)
	openwbnative.log.INFO.Printf("gpio_cp pin %d", gpio_cp)
	openwbnative.log.INFO.Printf("gpio_relay pin %d", gpio_relay)

	openwbnative.log.INFO.Println("Setting MaxCurrent to zero")
	if err := openwbnative.MaxCurrent(0); err != nil {
		return err
	}

	if err := rpio.Open(); err != nil {
		return err
	}

	// Unmap gpio memory when done
	defer rpio.Close()

	pin_gpio_cp := rpio.Pin(gpio_cp)
	pin_gpio_relay := rpio.Pin(gpio_relay)

	// Set pins to output mode
	pin_gpio_cp.Output()
	pin_gpio_relay.Output()

	openwbnative.log.INFO.Println("Sleeping for 1s")
	time.Sleep(time.Second)

	openwbnative.log.INFO.Println("CP off")
	pin_gpio_cp.High() // CP off

	openwbnative.log.INFO.Println("Toggle 1/3 relay high")
	pin_gpio_relay.High() // 3 on/off

	openwbnative.log.INFO.Printf("Sleeping for %d", seconds)
	time.Sleep(time.Second * time.Duration(seconds))

	openwbnative.log.INFO.Println("Toggle 1/3 relay low")
	pin_gpio_relay.Low() // 3 on off

	openwbnative.log.INFO.Printf("Sleeping for %d", seconds)
	time.Sleep(time.Second * time.Duration(seconds))

	openwbnative.log.INFO.Println("CP on")
	pin_gpio_cp.Low() // CP on

	openwbnative.log.INFO.Println("Sleeping for 1s")
	time.Sleep(time.Second)

	openwbnative.log.INFO.Println("Done with phase switch")

	return nil
}

func (openwbnative *OpenWbNative) perform_cp_interruption(seconds int) error {
	gpio_cp := get_pins_cp_interruption()
	openwbnative.log.INFO.Printf("gpio_cp pin %d", gpio_cp)

	openwbnative.log.INFO.Println("Setting MaxCurrent to zero")
	if err := openwbnative.MaxCurrent(0); err != nil {
		return err
	}

	if err := rpio.Open(); err != nil {
		return err
	}

	// Unmap gpio memory when done
	defer rpio.Close()

	pin_gpio_cp := rpio.Pin(gpio_cp)

	// Set pin to output mode
	pin_gpio_cp.Output()

	openwbnative.log.INFO.Println("Sleeping for 1s")
	time.Sleep(time.Second)

	openwbnative.log.INFO.Println("CP off")
	pin_gpio_cp.High()
	openwbnative.log.INFO.Printf("Sleeping for %d", seconds)
	time.Sleep(time.Second * time.Duration(seconds))
	openwbnative.log.INFO.Println("CP on")
	pin_gpio_cp.Low()
	openwbnative.log.INFO.Println("Done with cp interrupt")
	return nil
}

// CP Relay: Board 22, BCM/GPIO 25
// Switch to 1 Phase: Board 29 BCM/GPIO 5
// Switch to 3 Phase: Board 37 BCM/GPIO 26
func get_pins_phase_switch(new_phases int) (int, int) {
	// return gpio_cp, gpio_relay
	if new_phases == 1 {
		return 25, 5
	}
	return 25, 26
}

func get_pins_cp_interruption() int {
	// return gpio_cp
	return 25
}
