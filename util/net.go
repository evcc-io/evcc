package util

import (
	"fmt"
	"net"
)

// DefaultPort appends given port to connection if not specified
func DefaultPort(conn string, port int) string {
	if _, _, err := net.SplitHostPort(conn); err != nil {
		conn = fmt.Sprintf("%s:%d", conn, port)
	}

	return conn
}
