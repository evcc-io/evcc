package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrintPrice(t *testing.T) {
	for in, out := range map[float64]string{
		0.15:  "0.15",
		0.3:   "0.30",
		0.155: "0.155",
		-0.05: "-0.05",
		300:   "300.00",
	} {
		assert.Equal(t, out, printPrice(&in))
	}
	assert.Equal(t, "<nil>", printPrice(nil))
}
