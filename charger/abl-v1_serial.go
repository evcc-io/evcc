package charger

/*
This file is distributed under the MIT license
Copyright (c) 2026 marvinfourtytwo
*/

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// sendRawCommand composes the payload and sends it over serial line
func (wb *ABLv1) sendRawCommand(ctx context.Context, addr uint8, cmd uint8, payload string) error {
	var toSend string
	// if payload present, append a space and payload
	if len(payload) > 0 {
		if len(payload) != 4 {
			return fmt.Errorf("payload must be 4 bytes, got %d", len(payload))
		}
		toSend = fmt.Sprintf("!%d %02d %s\r\n", addr, cmd, payload)
	} else {
		toSend = fmt.Sprintf("!%d %02d\r\n", addr, cmd)
	}

	// log ASCII command being sent (as string)
	wb.log.TRACE.Printf("tx: %q", toSend)

	_, err := wb.conn.Write([]byte(toSend))

	return err
}

// readReply reads a single reply and validates it matches the expected addr and cmd
// charger may reply with excess spaces, so be careful here
func (wb *ABLv1) readReply(ctx context.Context, expectedAddr, expectedCmd uint8) (v int, err error) {
	var addr, cmd uint8
	var data string

	deadline := time.Now().Add(wb.rxTimeout)

	// read a line (without per-read timeout; this will block until data arrives)
	line, err := wb.r.ReadString('\n')
	// Log raw received ASCII line
	wb.log.TRACE.Printf("rx: %q", line)

	if err != nil {
		// if overall deadline exceeded, return timeout; otherwise continue to attempt until deadline
		if time.Now().After(deadline) {
			err = fmt.Errorf("timeout waiting for reply: %w", err)
		} else {
			err = fmt.Errorf("error during serial read: %w", err)
		}
		return 0, err
	}

	line = strings.TrimPrefix(line, ">")
	line = strings.TrimSuffix(line, "\r\n")

	// use fields here for robustnes, charger may return excess spaces
	f := strings.Fields(line)

	switch len(f) {
	case 2: // just ack
		_, err = fmt.Sscanf(line, "%d %d", &addr, &cmd)
		if err != nil {
			return 0, fmt.Errorf("error during scanf with two values: %w", err)
		}
	case 3: // return with value
		_, err = fmt.Sscanf(line, "%d %d %s", &addr, &cmd, &data)
		if err != nil {
			return 0, fmt.Errorf("error during scanf with three values: %w", err)
		}

		if data == "ERR" {
			return 0, fmt.Errorf("got error replay, command not allowed?")
		}
	default:
		return 0, fmt.Errorf("unexpected number of arguments returned")
	}

	if addr != expectedAddr {
		return 0, fmt.Errorf("reply from unknown wallbox received")
	}
	if cmd != expectedCmd {
		return 0, fmt.Errorf("reply for unknown command received")
	}

	if data != "" {
		v, err = strconv.Atoi(data)
		if err != nil {
			return 0, fmt.Errorf("Conversion to int failed: %w", err)
		}
	}

	return v, nil
}

// transact sends a command and waits for its reply
func (wb *ABLv1) transact(ctx context.Context, cmd uint8, payload string) (data int, err error) {
	wb.tr.Lock()
	defer wb.tr.Unlock()

	if err := wb.sendRawCommand(ctx, wb.addr, cmd, payload); err != nil {
		return 0, fmt.Errorf("error duing sent: %w", err)
	}

	data, err = wb.readReply(ctx, wb.addr, cmd)

	return data, err
}
