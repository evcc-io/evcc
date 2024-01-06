package util

import (
	"testing"
)

func TestCache(t *testing.T) {
	c := NewCache()

	c.Add("foo", Param{})
}
