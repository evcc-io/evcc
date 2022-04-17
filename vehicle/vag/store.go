package vag

import "github.com/evcc-io/evcc/util/store"

const (
	// IdkToken is the shared hash prefix for all IDK tokens
	IdkToken = "idk"
)

func StoreTokenProvider(store store.Store, key, rt string) func() (*Token, error) {
	return func() (*Token, error) {
		var token Token
		err := store.Get(key, &token.RefreshToken)

		// TODO
		token.RefreshToken = rt
		err = nil

		return &token, err
	}
}
