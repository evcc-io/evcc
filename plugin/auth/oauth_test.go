package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestOAuth(t *testing.T) {
	var storerCalled int
	storer := func(t *oauth2.Token) any {
		storerCalled++
		return t
	}

	ts, err := NewOAuth(t.Context(), "foo", "bar", &oauth2.Config{
		ClientID: "baz",
	}, WithTokenStorerOption(storer))
	require.NoError(t, err)

	token, err := ts.Token()
	require.ErrorContains(t, err, "login required")
	require.False(t, token.Valid())
	require.Equal(t, 0, storerCalled)

	ts.updateToken(&oauth2.Token{AccessToken: "at"})
	require.Equal(t, 1, storerCalled)

	token, err = ts.Token()
	require.NoError(t, err)
	require.True(t, token.Valid())
	require.Equal(t, 1, storerCalled)
}
