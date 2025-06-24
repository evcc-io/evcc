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

// OpenWbHw charger implementation
type OpenWbHw struct {
	conn        *modbus.Connection
	current     int64
	log         *util.Logger
	rfIdChannel chan string
	rfId        string
}

const (
	owbhwRegAmpsConfig    = 1000
	owbhwRegVehicleStatus = 1002

	owbhwGpioCP     = 25
	owbhwGpioRelay1 = 5
	owbhwGpioRelay3 = 26
)

func init() {
	registry.AddCtx("openwbhw", NewOpenWbHwFromConfig)
}

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

// GPIO 5 => 1 phasig, Schütz A an, Schütz B aus.
// GPIO 26 => 3 phasig, Schütz A und B an.
// Die CP-Unterbrechung wird über ein normales Relais mit NC auf BCM 25/Board 22 gesteuert.
// gpio=4,5,7,11,17,22,23,24,25,26,27=op,dl
// gpio=6,8,9,10,12,13,16,21=ip,pu

//go:generate go tool decorate -f decorateOpenWbHw -b *OpenWbHw -r api.Charger -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.Identifier,Identify,func() (string, error)"

// NewOpenWbHwFromConfig creates an OpenWbHw DIN charger from generic config
func NewOpenWbHwFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
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

	wb, err := NewOpenWbHw(ctx, cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.Protocol(), cc.ID)
	if err != nil {
		return nil, err
	}

	var phases1p3p func(int) error
	if cc.Phases1p3p {
		phases1p3p = wb.phases1p3p
	}

	if cc.RfId {
		log := util.NewLogger("openwbhw")
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
		wb.rfIdChannel = rfIdChannel
		for _, p := range keyboardPaths {
			go func(p string) {
				log.INFO.Printf("Monitoring keyboard %s\n", p)
				dev, err := evdev.Open(p)
				if err != nil {
					log.INFO.Printf("Cannot read %s: %v\n", p, err)
					return // Add this to exit the goroutine if device can't be opened
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
								log.INFO.Printf("Adding key \"%s\"", scanCodeMap[e.Code])
								if val, ok := scanCodeMap[e.Code]; ok {
									read += val
								} else {
									log.INFO.Printf("Unknown key code: %v", e.Code)
								}
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

	return decorateOpenWbHw(wb, phases1p3p, identify), nil
}

// NewOpenWbHw creates OpenWbHw charger
func NewOpenWbHw(ctx context.Context, uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8) (*OpenWbHw, error) {
	log := util.NewLogger("openwbhw")

	conn, err := modbus.NewConnection(ctx, uri, device, comset, baudrate, proto, slaveID)
	if err != nil {
		return nil, err
	}

	conn.Logger(log.TRACE)
	conn.Delay(200 * time.Millisecond)

	wb := &OpenWbHw{
		conn:    conn,
		current: 6, // assume min current
		log:     log,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *OpenWbHw) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(owbhwRegVehicleStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

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
func (wb *OpenWbHw) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(owbhwRegAmpsConfig, 1)
	if err != nil {
		return false, err
	}

	enabled := b[1] != 0
	if enabled {
		wb.current = int64(b[1])
	}

	return enabled, nil
}

// Enable implements the api.Charger interface
func (wb *OpenWbHw) Enable(enable bool) error {
	b := []byte{0, 0}

	if enable {
		b[1] = byte(wb.current)
	}

	_, err := wb.conn.WriteMultipleRegisters(owbhwRegAmpsConfig, 1, b)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *OpenWbHw) MaxCurrent(current int64) error {
	b := []byte{0, byte(current)}

	_, err := wb.conn.WriteMultipleRegisters(owbhwRegAmpsConfig, 1, b)
	if err == nil {
		wb.current = current
	}

	return err
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *OpenWbHw) phases1p3p(phases int) error {
	if err := wb.Enable(false); err != nil {
		return err
	}
	time.Sleep(time.Second)

	if err := rpio.Open(); err != nil {
		return err
	}

	// Unmap gpio memory when done
	defer rpio.Close()

	pinGpioCP := rpio.Pin(owbhwGpioCP)
	pinGpioPhases := rpio.Pin(owbhwGpioRelay3)
	if phases == 1 {
		pinGpioPhases = rpio.Pin(owbhwGpioRelay1)
	}

	// Set pins to output mode
	pinGpioCP.Output()
	pinGpioPhases.Output()

	pinGpioCP.High() // enable CP disconnect relay
	time.Sleep(time.Second)
	pinGpioPhases.High() // move latching relay to desired position
	time.Sleep(time.Second)
	pinGpioPhases.Low() // lock latching relay
	time.Sleep(time.Second)
	pinGpioCP.Low() // disable CP disconnect relay
	time.Sleep(time.Second)

	return nil
}

var _ api.Resurrector = (*OpenWbHw)(nil)

// WakeUp implements the api.Resurrector interface
func (wb *OpenWbHw) WakeUp() error {
	if err := wb.Enable(false); err != nil {
		return err
	}

	if err := rpio.Open(); err != nil {
		return err
	}

	// Unmap gpio memory when done
	defer rpio.Close()

	gpioPinCP := rpio.Pin(owbhwGpioCP)

	// Set pin to output mode
	gpioPinCP.Output()

	gpioPinCP.High()
	time.Sleep(time.Second * time.Duration(3))
	gpioPinCP.Low()

	return nil
}

// Identify implements the api.Identifier interface
func (wb *OpenWbHw) identify() (string, error) {
	var completed bool = false

	for !completed {
		select {
		case rfid := <-wb.rfIdChannel:
			wb.log.INFO.Printf("Read RFID \"%s\" from channel", rfid)
			wb.rfId = rfid
		default:
			wb.log.INFO.Println("Nothing left to read from channel")
			completed = true
		}
	}

	return wb.rfId, nil
}
