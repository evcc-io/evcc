package vag

import (
	"github.com/evcc-io/evcc/util/store"
)

const (
	// IdkToken is the shared hash prefix for all IDK tokens
	IdkToken = "idk"
)

type RefreshTokenStore interface {
	Get() (*Token, error)
	Put(*Token) error
}

func StoreTokenProvider(store store.Store, key, rt string) RefreshTokenStore {
	return &stp{
		store: store,
		key:   key,
		rt:    rt,
	}
}

type stp struct {
	store store.Store
	key   string
	rt    string
}

func (s *stp) Get() (*Token, error) {
	var token Token
	err := s.store.Get(s.key, &token.RefreshToken)

	// TODO
	// token.RefreshToken = s.rt
	// err = nil

	return &token, err
}

func (s *stp) Put(token *Token) error {
	return s.store.Put(s.key, token.RefreshToken)
}
