//go:build linux || darwin || freebsd || openbsd || netbsd || dragonfly

package batterycontrol

import (
	"os"

	"golang.org/x/sys/unix"
)

func lockJournal(file *os.File) error { return unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB) }
