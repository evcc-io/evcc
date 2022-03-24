package vag

import (
	"testing"

	"golang.org/x/oauth2"
)

func TestMerge(t *testing.T) {
	ts := &tokenSource{
		token: &Token{
			IDToken: "id",
			Token: oauth2.Token{
				AccessToken:  "access",
				RefreshToken: "refresh",
			},
		},
	}

	r := &Token{
		IDToken: "foo",
		Token: oauth2.Token{
			AccessToken: "bar",
		},
	}

	if err := ts.mergeToken(r); err != nil {
		t.Error(err)
	}

	if ts.token.IDToken != "foo" {
		t.Error("unexpected id token", ts.token)
	}
	if ts.token.AccessToken != "bar" {
		t.Error("unexpected access token", ts.token)
	}
	if ts.token.RefreshToken != "refresh" {
		t.Error("unexpected refresh token", ts.token)
	}
}
