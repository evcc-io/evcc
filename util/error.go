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
		LoginRequired string `json:"loginRequired,omitempty"`
		URI           string `json:"uri,omitempty"`
	}{
		Error: err.Error(),
	}

	if ae := new(api.ErrLoginRequired); errors.As(err, &ae) {
		res.LoginRequired = ae.ProviderAuth
	}

	if ue := new(api.ErrUrl); errors.As(err, &ue) {
		res.URI = ue.URL().String()
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
