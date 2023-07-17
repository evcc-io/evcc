package oauth

import (
	"time"

	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

// Refresh refreshes the token every 5m. If token refresh fails 5 times, it is aborted.
func Refresh(log *util.Logger, token *oauth2.Token, ts oauth2.TokenSource, optMaxTokenLifetime ...time.Duration) {
	var failed int

	for range time.Tick(5 * time.Minute) {
		if _, err := ts.Token(); err != nil {
			t, err := ts.Token()
			if err != nil {
				failed++
				if failed > 5 {
					log.ERROR.Printf("token refresh: %v, giving up", err)
					return
				}
				log.ERROR.Printf("token refresh: %v", err)
			}

			failed = 0

			// limit lifetime of new tokens
			if len(optMaxTokenLifetime) == 1 && t.Expiry != token.Expiry {
				token = t
				maxTokenLifetime := optMaxTokenLifetime[0]
				if time.Until(token.Expiry) > maxTokenLifetime {
					token.Expiry = time.Now().Add(maxTokenLifetime)
				}
			}
		}
	}
}
