//go:build !linux && !darwin

package discovery

import (
	"context"
	"os/exec"
	"runtime"
	"time"
)

func readNeighbors() (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// numeric output, name resolution is slow
	args := "-an"
	if runtime.GOOS == "windows" {
		args = "-a"
	}

	b, err := exec.CommandContext(ctx, "arp", args).Output()
	return parseNeighbors(string(b)), err
}
