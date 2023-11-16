package sponsor

import (
	"os"
	"time"
)

func readSerial() (string, error) {
	f, err := os.OpenFile("/dev/PulsaresSerial", os.O_RDWR, 0o644)
	if err != nil {
		return "", nil
	}

	if _, err := f.Write([]byte{0x0E, 0x00, 0xD0, 0x05}); err != nil {
		return "", err
	}

	// serial timeout
	time.AfterFunc(3*time.Second, func() {
		_ = f.Close()
	})

	b := make([]byte, 512)
	n, _ := f.Read(b)

	return string(b[:n]), nil
}
