package modbus

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
)

func Backoff() *backoff.ExponentialBackOff {
	return backoff.NewExponentialBackOff(backoff.WithInitialInterval(20*time.Millisecond), backoff.WithMaxElapsedTime(10*time.Second))
}

// decodeMask converts a bit mask in decimal or hex format to uint64
func decodeMask(mask string) (uint64, error) {
	mask = strings.ToLower(mask)

	if mask == "" {
		return 0, errors.New("mask is required")
	}

	if strings.HasPrefix(mask, "0x") {
		if len(mask) < 3 {
			return 0, fmt.Errorf("invalid mask: %s", mask)
		}

		b, err := hex.DecodeString(mask[2:])
		if err != nil {
			return 0, fmt.Errorf("invalid mask: %w", err)
		}

		var u uint64
		for _, v := range b {
			u = u<<8 | uint64(v)
		}

		return u, nil
	}

	return strconv.ParseUint(mask, 10, 64)
}

// decodeBool8 converts a masked uint1 to a bool
func decodeBool8(b []byte) float64 {
	u := b[0]
	if u > 0 {
		return 1
	}
	return 0
}

// decodeBool16 converts a masked uint16 to a bool
func decodeBool16(mask uint64) func(b []byte) float64 {
	return func(b []byte) float64 {
		u := binary.BigEndian.Uint16(b)
		if mask > 0 {
			u &= uint16(mask)
		}
		if u > 0 {
			return 1
		}
		return 0
	}
}

func decodeNaN16(f func(b []byte) float64, nan ...uint16) func(b []byte) float64 {
	return func(b []byte) float64 {
		u := binary.BigEndian.Uint16(b)
		if slices.Contains(nan, u) {
			return 0
		}
		return f(b)
	}
}

func decodeNaN32(f func(b []byte) float64, nan ...uint32) func(b []byte) float64 {
	return func(b []byte) float64 {
		u := binary.BigEndian.Uint32(b)
		if slices.Contains(nan, u) {
			return 0
		}
		return f(b)
	}
}

func decodeNaN64(f func(b []byte) float64, nan ...uint64) func(b []byte) float64 {
	return func(b []byte) float64 {
		u := binary.BigEndian.Uint64(b)
		if slices.Contains(nan, u) {
			return 0
		}
		return f(b)
	}
}
