package oauth

import (
	"testing"

	"golang.org/x/oauth2"
)

func TestMerge(t *testing.T) {
	ts := &TokenSource{
		token: &oauth2.Token{
			AccessToken:  "access",
			RefreshToken: "refresh",
		},
	}

	r := &oauth2.Token{
		AccessToken: "new",
	}

	ts.mergeToken(r)

	if ts.token.AccessToken != "new" {
		t.Error("unexpected access token", ts.token)
	}
	if ts.token.RefreshToken != "refresh" {
		t.Error("unexpected refresh token", ts.token)
	}
}
