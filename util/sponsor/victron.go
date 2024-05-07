package sponsor

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

func isVictron() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/usr/bin/board-compat")
	b, _ := cmd.Output()

	return strings.HasPrefix(string(b), "victronenergy,")
}
