package request

import (
	"errors"

	"github.com/cenkalti/backoff/v4"
)

func BackoffDefaultHttpStatusCodesPermanently() func(error) error {
	return BackoffHttpStatusCodesPermanently(DefaultPermanentBackoffHttpStatusCodes())
}

type HttpStatusCodeIsPermanentErr func(statusCode int) bool

func BackoffHttpStatusCodesPermanently(isPermanentErr HttpStatusCodeIsPermanentErr) func(error) error {
	return func(err error) error {
		se := new(StatusError)
		if errors.As(err, &se) {
			code := se.StatusCode()
			if isPermanentErr(code) {
				return backoff.Permanent(se)
			} else {
				return err
			}
		}
		return err
	}
}

func DefaultPermanentBackoffHttpStatusCodes() HttpStatusCodeIsPermanentErr {
	return PermanentBackoffHttpStatusCodeRange(400, 599)
}

func PermanentBackoffHttpStatusCodeRange(lowerBound int, upperBound int) HttpStatusCodeIsPermanentErr {
	return func(statusCode int) bool {
		if statusCode >= lowerBound && statusCode <= upperBound {
			return true
		}
		return false
	}
}

func TemporaryBackoffHttpStatusCode(tempErrStatusCode int, fallback HttpStatusCodeIsPermanentErr) HttpStatusCodeIsPermanentErr {
	return func(statusCode int) bool {
		if statusCode == tempErrStatusCode {
			return false
		}
		return fallback(statusCode)
	}
}
