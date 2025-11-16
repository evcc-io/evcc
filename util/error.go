package util

import (
	"errors"

	"github.com/evcc-io/evcc/api"
	"go.yaml.in/yaml/v4"
)

// ErrorAsJson returns an error as json-formattable struct
func ErrorAsJson(err error) any {
	res := struct {
		Error         string `json:"error"`
		Line          int    `json:"line,omitempty"`
		IsAuthError   bool   `json:"isAuthError,omitempty"`
		LoginRequired string `json:"loginRequired,omitempty"`
	}{
		Error:       err.Error(),
		IsAuthError: errors.Is(err, api.ErrMissingToken),
	}

	if ae := new(api.ErrLoginRequired); errors.As(err, &ae) {
		res.IsAuthError = true
		res.LoginRequired = ae.ProviderAuth
	}

	var (
		ype *yaml.ParserError
		yue *yaml.UnmarshalError
	)
	switch {
	case errors.As(err, &ype):
		res.Line = ype.Line
	case errors.As(err, &yue):
		res.Line = yue.Line
	}

	return res
}
