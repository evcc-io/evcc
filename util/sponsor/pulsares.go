package sponsor

import (
	"os"
	"strings"
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

	var token string
	b := make([]byte, 512)

	for {
		n, err := f.Read(b)
		if err != nil {
			return "", nil
		}

		token += string(b[:n])

		if token, ok := strings.CutSuffix(token, "\x04"); ok {
			return token, nil
		}
	}
}
