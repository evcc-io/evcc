package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/cryptobyte"
)

func TestPatch(t *testing.T) {
	var res bool
	b := cryptobyte.String([]byte{0x01, 0x01, 0x01})
	ok := b.ReadASN1Boolean(&res)
	assert.True(t, ok, "read failed")
	assert.Equal(t, true, res, "patch failed")
}
