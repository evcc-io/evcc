package request

import (
	"errors"

	"github.com/cenkalti/backoff/v4"
)

func BackoffDefaultHttpStatusCodesPermanently(options ...HttpBackoffOption) func(error) error {
	// defaults will be matched last
	return BackoffHttpStatusCodesPermanently(append(options, BackoffHttpStatusCodeRange(400, 599, true))...)
}

func BackoffHttpStatusCodesPermanently(options ...HttpBackoffOption) func(error) error {
	return func(err error) error {
		se := new(StatusError)
		if errors.As(err, &se) {
			code := se.StatusCode()
			for _, option := range options {
				backoffResult := option.BackoffForStatusCode(code)
				if backoffResult != nil {
					if *backoffResult {
						return backoff.Permanent(se)
					} else {
						return err
					}
				}
			}
		}
		return err
	}
}

type HttpBackoffOption interface {
	BackoffForStatusCode(statusCode int) *bool
}

func BackoffHttpStatusCodeRange(lowerBound int, upperBound int, permanentErr bool) HttpBackoffOption {
	return httpBackoffStatusCodeRange{lowerBound, upperBound, permanentErr}
}

func BackoffHttpStatusCode(code int, permanentErr bool) HttpBackoffOption {
	return BackoffHttpStatusCodeRange(code, code, permanentErr)
}

type httpBackoffStatusCodeRange struct {
	lowerBound, upperBound int
	permanentErr           bool
}

func (h httpBackoffStatusCodeRange) BackoffForStatusCode(statusCode int) *bool {
	if statusCode >= h.lowerBound && statusCode <= h.upperBound {
		return &h.permanentErr
	}
	return nil
}
