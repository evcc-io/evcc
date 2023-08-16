package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func TestFormatValue(t *testing.T) {
	mp := message.NewPrinter(language.Make("en"))

	f := 1.2345
	assert.Equal(t, "1.234", formatValue(mp, f, 3))
	assert.Equal(t, "1.234", formatValue(mp, &f, 3))
}
