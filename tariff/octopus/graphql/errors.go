package graphql

import (
	"errors"
)

var (
	ErrAccountNotFound  = errors.New("unable to find configured account")
	ErrMultipleAccounts = errors.New("multiple accounts on this api key - specific an account to use in configuration")
	ErrNoAccounts       = errors.New("no accounts on this api key")
)
