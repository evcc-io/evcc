package oauth

import (
	"sync"

	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

type redactedTokenSource struct {
	mu     sync.Mutex
	ts     oauth2.TokenSource
	access string
	redact func(...string)
}

// Redacted keeps the source's access and refresh tokens out of the logs, where
// they would otherwise show up in the Authorization header. The tokens share a
// rotating redaction slot, so refreshing replaces them instead of growing the
// redaction list.
func Redacted(log *util.Logger, ts oauth2.TokenSource) oauth2.TokenSource {
	return &redactedTokenSource{
		ts:     ts,
		redact: log.RotatingSlot(),
	}
}

func (ts *redactedTokenSource) Token() (*oauth2.Token, error) {
	token, err := ts.ts.Token()
	if err != nil {
		return token, err
	}

	ts.mu.Lock()
	defer ts.mu.Unlock()

	if token.AccessToken != ts.access {
		ts.access = token.AccessToken
		ts.redact(token.AccessToken, token.RefreshToken)
	}

	return token, nil
}
