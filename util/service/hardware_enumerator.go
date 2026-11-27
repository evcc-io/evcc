//go:build !darwin || cgo

package service

import (
	"strings"

	"github.com/evcc-io/evcc/server/service"
	"go.bug.st/serial/enumerator"
)

// serialPortOptions lists ports with USB product, vid:pid and serial number
func serialPortOptions() []service.Option {
	details, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return plainSerialPorts()
	}

	res := make([]service.Option, 0, len(details))
	for _, d := range details {
		// match on every port so a single port still auto-fills
		o := service.Option{Value: d.Name, Label: d.Product, Match: true}
		if d.VID != "" {
			o.Hint = strings.TrimSpace(d.VID + ":" + d.PID + " " + d.SerialNumber)
		}
		res = append(res, o)
	}
	return res
}
