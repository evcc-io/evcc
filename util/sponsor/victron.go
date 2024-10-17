package sponsor

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api/proto/pb"
	"github.com/evcc-io/evcc/util/cloud"
	"github.com/evcc-io/evcc/util/request"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// checkVictron checks if the hardware is a supported victron device and returns sponsor subject
func checkVictron() string {
	vd, err := victronDeviceInfo()
	if err != nil {
		// unable to retrieve all device info
		return ""
	}

	conn, err := cloud.Connection()
	if err != nil {
		// unable to connect to cloud
		return unavailable
	}

	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

	client := pb.NewVictronClient(conn)
	res, err := client.IsValidDevice(ctx, &pb.VictronRequest{
		ProductId: vd.ProductId,
		VrmId:     vd.VrmId,
		Serial:    vd.Serial,
		Board:     vd.Board,
	})

	if err == nil && res.Authorized {
		// cloud check successful
		return victron
	}

	if s, ok := status.FromError(err); ok && s.Code() != codes.Unknown {
		// technical error during validation
		return unavailable
	}

	return ""
}

type victronDevice struct {
	ProductId string
	VrmId     string
	Serial    string
	Board     string
}

func commandExists(cmd string) error {
	_, err := exec.LookPath(cmd)
	return err
}

func executeCommand(ctx context.Context, cmd string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, cmd, args...)
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	result := strings.TrimSpace(string(output))
	if result == "" {
		return "", errors.New("empty output")
	}
	return result, nil
}

func victronDeviceInfo() (victronDevice, error) {
	if runtime.GOOS != "linux" {
		return victronDevice{}, errors.New("non-linux os")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var vd victronDevice

	commands := []struct {
		field *string
		cmd   string
		args  []string
	}{
		{field: &vd.Board, cmd: "/usr/bin/board-compat"},
		{field: &vd.ProductId, cmd: "/usr/bin/product-id"},
		{field: &vd.VrmId, cmd: "/sbin/get-unique-id"},
		{field: &vd.Serial, cmd: "/opt/victronenergy/venus-eeprom/eeprom", args: []string{"--show", "serial-number"}},
	}

	for _, detail := range commands {
		if err := commandExists(detail.cmd); err != nil {
			return vd, errors.New("cmd not found: " + detail.cmd)
		}
		output, err := executeCommand(ctx, detail.cmd, detail.args...)
		if err != nil {
			return vd, err
		}
		*detail.field = output
	}

	return vd, nil
}
