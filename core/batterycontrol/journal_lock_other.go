//go:build !linux && !darwin && !freebsd && !openbsd && !netbsd && !dragonfly && !windows

package batterycontrol

import (
	"fmt"
	"os"
)

func lockJournal(*os.File) error {
	return fmt.Errorf("persistent battery control not supported on this platform")
}
