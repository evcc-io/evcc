package oauth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type tkr struct{}

func (tkr *tkr) RefreshToken(_ *oauth2.Token) (*oauth2.Token, error) {
	return (&oauth2.Token{
		AccessToken: "new",
	}).WithExtra(map[string]any{
		"foo": "bar",
	}), nil
}

func TestMerge(t *testing.T) {
	ts := &TokenSource{
		token: &oauth2.Token{
			AccessToken:  "access",
			RefreshToken: "refresh",
			Expiry:       time.Now(),
		},
		refresher: new(tkr),
	}

	r, err := ts.Token()
	require.NoError(t, err)

	require.Equal(t, "new", r.AccessToken, "unexpected access token")
	require.Equal(t, "refresh", r.RefreshToken, "unexpected refresh token")
	require.Equal(t, "bar", r.Extra("foo"))
}
