package request

import (
	"fmt"
	"net/url"
	"strings"
)

// URLValue is a single URLValues entry
type URLValue struct {
	K string
	V interface{}
}

// URLValues provides url.Values functionality while keeping parameter sort order
type URLValues []URLValue

// NewURLValues creates URLValues
func NewURLValues(values []URLValue) *URLValues {
	u := make(URLValues, 0, len(values))
	u = append(u, values...)
	return &u
}

// Encode encodes the URLValues into string
func (u *URLValues) Encode() string {
	res := strings.Builder{}
	for _, e := range *u {
		if res.Len() > 0 {
			res.WriteRune('&')
		}

		res.WriteString(e.K)
		res.WriteRune('=')

		val := fmt.Sprintf("%s", e.V)
		res.WriteString(url.QueryEscape(val))
	}

	return res.String()
}
